// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure JiraProvider satisfies various provider interfaces.
var _ provider.Provider = &JiraProvider{}
var _ provider.ProviderWithValidateConfig = &JiraProvider{}

// JiraProvider defines the provider implementation.
type JiraProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// client is the Jira client.
	client *jira.Client
	// provider-level operation timeouts
	providerTimeouts opTimeouts
}

// JiraProviderModel describes the provider data model.
type JiraProviderModel struct {
	// Base Configuration
	Endpoint types.String `tfsdk:"endpoint"`

	AuthMethod types.String `tfsdk:"auth_method"`

	// Logging
	Debug types.Bool `tfsdk:"debug"`

	OperationTimeouts *OperationTimeoutsModel `tfsdk:"operation_timeouts"`

	// HTTP Settings
	HTTPTimeoutSeconds    types.Int64 `tfsdk:"http_timeout_seconds"`
	RetryOn4295xx         types.Bool  `tfsdk:"retry_on_429_5xx"`
	RetryMaxAttempts      types.Int64 `tfsdk:"retry_max_attempts"`
	RetryInitialBackoffMs types.Int64 `tfsdk:"retry_initial_backoff_ms"`
	RetryMaxBackoffMs     types.Int64 `tfsdk:"retry_max_backoff_ms"`

	// API Token Authentication
	APIToken     types.String `tfsdk:"api_token"`
	APIAuthEmail types.String `tfsdk:"api_auth_email"`

	// Basic Authentication
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`

	// Privacy & Redaction
	EmailRedactionMode types.String `tfsdk:"email_redaction_mode"`
}

// Metadata sets the provider type name and version for Terraform.
func (j *JiraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jira"
	resp.Version = j.version
}

// Schema returns the provider schema describing configuration attributes.
func (j *JiraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Jira provider for interacting with Jira instances using the go-jira library.",
		Attributes: map[string]schema.Attribute{
			// Base Configuration
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base Endpoint of the Jira client (e.g., 'https://your-domain.atlassian.net'). Can be set with environment variable `JIRA_ENDPOINT` (canonical) or alias `JIRA_BASE_URL`. Precedence: provider attributes > canonical env var > alias.",
				Optional:            true,
			},

			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method to use for Jira. Default: \"api_token\". Accepts values `api_token` or `basic`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`(api_token|basic|^$)`), "auth_method must be one of 'api_token' or 'basic'."),
				},
			},

			// Logging
			"debug": schema.BoolAttribute{
				MarkdownDescription: "Enable additional provider debug logs. Honors TF_LOG for log level; when true, the provider emits extra structured debug logs with sensitive values redacted.",
				Optional:            true,
			},

			"http_timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "HTTP client timeout in seconds for all Jira API requests. Defaults to 30 seconds. Acceptable range is 1–600. Rationale: 0 disables the Go net/http client timeout and risks hung plans; a minimum of 1 second avoids indefinite waits. The 600-second (10 minute) maximum caps a single HTTP attempt to prevent runaway applies and aligns with typical upstream gateway/service limits. For long-running operations, prefer per-operation timeouts via operation_timeouts and consider retry/backoff settings—overall wall time includes (retries + 1) × http_timeout_seconds plus backoff.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 600),
				},
			},

			// Retry Settings
			"retry_on_429_5xx": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic retries on HTTP 429 and 5xx responses. Defaults to true.",
				Optional:            true,
			},
			"retry_max_attempts": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of retry attempts for transient failures. Defaults to 4. Allowed range: 1–10.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"retry_initial_backoff_ms": schema.Int64Attribute{
				MarkdownDescription: "Initial backoff, in milliseconds, before the first retry. Defaults to 500 ms. Allowed range: 100–600000.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(100, 600000),
				},
			},
			"retry_max_backoff_ms": schema.Int64Attribute{
				MarkdownDescription: "Maximum backoff, in milliseconds, for retries. Defaults to 5000 ms. Allowed range: 100–600000.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(100, 600000),
				},
			},

			// API Token Authentication (Recommended for Jira Cloud)
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token (PAT) for authentication. **Required** when using API token authentication with email.Can be set with environment variable `JIRA_API_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("api_auth_email")),
					stringvalidator.ConflictsWith(path.MatchRoot("username")),
					stringvalidator.ConflictsWith(path.MatchRoot("password")),
				},
			},
			"api_auth_email": schema.StringAttribute{
				MarkdownDescription: "Email address associated with the API token. **Required** when using API token authentication. Can be set with environment variable `JIRA_API_EMAIL` (canonical) or alias `JIRA_EMAIL`. Precedence: provider attributes > canonical env var > alias.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("api_token")),
					stringvalidator.ConflictsWith(path.MatchRoot("username")),
					stringvalidator.ConflictsWith(path.MatchRoot("password")),
				},
			},

			// Basic Authentication (For self-hosted Jira)
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for basic authentication. **Required** when using basic authentication with password.Can be set with environment variable `JIRA_USERNAME`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("password")),
					stringvalidator.ConflictsWith(path.MatchRoot("api_token")),
					stringvalidator.ConflictsWith(path.MatchRoot("api_auth_email")),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for basic authentication. **Required** when using basic authentication.Can be set with environment variable `JIRA_PASSWORD`.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("username")),
					stringvalidator.ConflictsWith(path.MatchRoot("api_token")),
					stringvalidator.ConflictsWith(path.MatchRoot("api_auth_email")),
				},
			},
			"operation_timeouts": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional per-operation timeouts for provider-managed operations. Use Go duration strings like '30s', '2m', '1h'. Each value must be greater than 0 if set.",
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Timeout for create operations. Example: '2m'.",
					},
					"read": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Timeout for read operations. Example: '30s'.",
					},
					"update": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Timeout for update operations. Example: '2m'.",
					},
					"delete": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Timeout for delete operations. Example: '2m'.",
					},
				},
			},

			// Privacy & Redaction
			"email_redaction_mode": schema.StringAttribute{
				MarkdownDescription: "Controls how emails are sanitized in logs/errors. Default: \"full\". Allowed values: `full` (fully redact as \"[REDACTED_EMAIL]\") or `mask` (partially mask local-part and keep domain, e.g., \"a****@example.com\"). Can be set via environment variable `JIRA_EMAIL_REDACTION_MODE`. Precedence: provider attribute > env var.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`(full|mask|^$)`), "email_redaction_mode must be one of 'full' or 'mask'."),
				},
			},
		},
	}
}

func (j *JiraProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data JiraProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rc := deriveResolvedConfig(data)
	verrs := validateResolvedConfig(rc)
	if len(verrs) > 0 {
		for _, e := range verrs {
			if e.attr != "" {
				resp.Diagnostics.AddAttributeError(path.Root(e.attr), e.summary, e.detail)
			} else {
				resp.Diagnostics.AddError(e.summary, e.detail)
			}
		}
		return
	}

	// Validate operation_timeouts block (if provided)
	if data.OperationTimeouts != nil {
		_, verrs := parseOperationTimeouts(data.OperationTimeouts)
		if len(verrs) > 0 {
			for _, e := range verrs {
				resp.Diagnostics.AddAttributeError(path.Root("operation_timeouts").AtName(e.attr), e.summary, e.detail)
			}
			return
		}
	}
}

// Configure establishes the Jira API client and validates authentication.
// It attaches the configured client to resources and data sources for use in operations.
func (j *JiraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data JiraProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Setup debug logging context
	ctx, isDebug := j.setupDebug(ctx, data)

	// Resolve and validate configuration
	rc, verrs := j.resolveAndValidateConfig(ctx, data, isDebug)
	if len(verrs) > 0 {
		for _, e := range verrs {
			if e.attr != "" {
				resp.Diagnostics.AddAttributeError(path.Root(e.attr), e.summary, e.detail)
			} else {
				resp.Diagnostics.AddError(e.summary, e.detail)
			}
		}
		return
	}

	// Parse and assign provider-level operation timeouts
	if data.OperationTimeouts != nil {
		pOT, otErrs := parseOperationTimeouts(data.OperationTimeouts)
		if len(otErrs) > 0 {
			for _, e := range otErrs {
				resp.Diagnostics.AddAttributeError(path.Root("operation_timeouts").AtName(e.attr), e.summary, e.detail)
			}
			return
		}
		j.providerTimeouts = pOT
	}

	// Initialize HTTP client
	httpClient := buildHTTPClient(rc)

	// Initialize Jira client with auth and user agent
	client, err := j.initJiraClient(httpClient, rc)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root(attrEndpoint), "Error creating Jira client", RedactSecrets(err.Error()))
		return
	}

	// Test the connection
	if !j.testConnection(ctx, client, &resp.Diagnostics) {
		return
	}

	// Attach clients for resources/data sources
	j.client = client
	resp.ResourceData = j
	resp.DataSourceData = j
}

// Resources returns the set of Terraform resources supported by the provider.
func (j *JiraProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkTypeResource,
		NewWorkflowStatusResource,
		NewProjectResource,
		NewProjectCategoryResource,
		NewFieldResource,
	}
}

// DataSources returns the set of Terraform data sources supported by the provider.
func (j *JiraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewWorkTypesDataSource,
		NewProjectDataSource,
		NewProjectsDataSource,
		NewProjectCategoriesDataSource,
	}
}

// New returns a constructor that builds a JiraProvider with the given version string.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &JiraProvider{
			version: version,
		}
	}
}

// centralized config structures and helpers

// Centralized provider defaults for visibility and reuse

// sanitizeEmail masks or redacts an email to avoid PII leakage in logs/errors.
// Mode controls behavior:
//   - "full": fully redact, returning "[REDACTED_EMAIL]"
//   - "mask": partially mask local-part, preserving domain (e.g., "a****@example.com")

// configuration derivation (unified) to avoid duplicated parsing across sections

// validation per-section

// setupDebug configures minimal safe tflog fields and returns whether debug is enabled.
func (j *JiraProvider) setupDebug(ctx context.Context, data JiraProviderModel) (context.Context, bool) {
	isDebug := false
	if !data.Debug.IsNull() && !data.Debug.IsUnknown() {
		isDebug = data.Debug.ValueBool()
	}
	if isDebug {
		ctx = tflog.SetField(ctx, "provider", "jira")
		ctx = tflog.SetField(ctx, "version", j.version)
		tflog.Info(ctx, "Provider debug enabled. Set TF_LOG=DEBUG to view detailed logs.")
	}
	return ctx, isDebug
}

// resolveAndValidateConfig derives resolved config, optionally logs safe fields, and validates.
func (j *JiraProvider) resolveAndValidateConfig(ctx context.Context, data JiraProviderModel, isDebug bool) (resolvedConfig, []validationErr) {
	rc := deriveResolvedConfig(data)
	if isDebug {
		tflog.Debug(ctx, "Resolved non-sensitive provider settings", map[string]interface{}{
			"auth_method":              rc.authMethod,
			"http_timeout_seconds":     rc.httpTimeoutSeconds,
			"retry_on_429_5xx":         rc.retryOn4295xx,
			"retry_max_attempts":       rc.retryMaxAttempts,
			"retry_initial_backoff_ms": rc.retryInitialBackoffMs,
			"retry_max_backoff_ms":     rc.retryMaxBackoffMs,
			"email_redaction_mode":     rc.emailRedactionMode,
		})
	}
	verrs := validateResolvedConfig(rc)
	return rc, verrs
}
