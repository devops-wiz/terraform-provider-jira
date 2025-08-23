// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	}
}

// DataSources returns the set of Terraform data sources supported by the provider.
func (j *JiraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewWorkTypesDataSource,
		NewProjectDataSource,
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

const (
	attrEndpoint            = "endpoint"
	attrAuthMethod          = "auth_method"
	attrAPIToken            = "api_token"
	attrAPIAuthEmail        = "api_auth_email"
	attrUsername            = "username"
	attrPassword            = "password"
	attrHTTPTimeoutSeconds  = "http_timeout_seconds"
	attrRetryOn4295xx       = "retry_on_429_5xx"
	attrRetryMaxAttempts    = "retry_max_attempts"
	attrRetryInitialBackoff = "retry_initial_backoff_ms"
	attrRetryMaxBackoff     = "retry_max_backoff_ms"
	attrEmailRedactionMode  = "email_redaction_mode"
)

// Centralized provider defaults for visibility and reuse
const (
	defaultAuthMethod            = "api_token"
	defaultHTTPTimeoutSeconds    = 30
	defaultRetryOn4295xx         = true
	defaultRetryMaxAttempts      = 4
	defaultRetryInitialBackoffMs = 500
	defaultRetryMaxBackoffMs     = 5000
	defaultEmailRedactionMode    = "full"
)

type validationErr struct {
	attr    string // empty for general error
	summary string
	detail  string
}

type resolvedConfig struct {
	endpoint              string
	authMethod            string
	email                 string
	apiToken              string
	username              string
	password              string
	httpTimeoutSeconds    int
	retryOn4295xx         bool
	retryMaxAttempts      int
	retryInitialBackoffMs int
	retryMaxBackoffMs     int
	emailRedactionMode    string
}

// sanitizeEmail masks or redacts an email to avoid PII leakage in logs/errors.
// Mode controls behavior:
//   - "full": fully redact, returning "[REDACTED_EMAIL]"
//   - "mask": partially mask local-part, preserving domain (e.g., "a****@example.com")
func sanitizeEmail(email string, mode string) string {
	if email == "" {
		return ""
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = defaultEmailRedactionMode
	}
	if mode != "full" && mode != "mask" {
		mode = defaultEmailRedactionMode
	}
	// Full redaction: never expose local-part or domain.
	if mode == "full" {
		return "[REDACTED_EMAIL]"
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		// Not a standard email; return a generic token to avoid echoing the raw value.
		return "[REDACTED_EMAIL]"
	}
	domain := email[at+1:]
	return "a****@" + domain
}

// redactSecretValue replaces a sensitive value with a stable token.
// If the value is empty, it returns the empty string to avoid adding tokens where not needed.
func redactSecretValue(v string) string {
	if v == "" {
		return ""
	}
	return "[REDACTED]"
}

// sanitizeValidationError returns a copy of the given validation error with secrets redacted.
func sanitizeValidationError(e validationErr, rc resolvedConfig) validationErr {
	// Build a mapping of raw -> redacted tokens
	replacements := map[string]string{}

	if rc.apiToken != "" {
		replacements[rc.apiToken] = redactSecretValue(rc.apiToken)
	}
	if rc.password != "" {
		replacements[rc.password] = redactSecretValue(rc.password)
	}
	if rc.username != "" {
		// Usernames can be sensitive; redact fully.
		replacements[rc.username] = redactSecretValue(rc.username)
	}
	if rc.email != "" {
		// For emails, replace with policy-driven sanitization (defaults to full).
		replacements[rc.email] = sanitizeEmail(rc.email, rc.emailRedactionMode)
	}

	// Apply replacements to summary/detail without echoing the original values.
	summary := e.summary
	detail := e.detail
	for raw, red := range replacements {
		if raw == "" {
			continue
		}
		if summary != "" {
			summary = strings.ReplaceAll(summary, raw, red)
		}
		if detail != "" {
			detail = strings.ReplaceAll(detail, raw, red)
		}
	}

	e.summary = summary
	e.detail = detail
	return e
}

// generic readers (HCL over env, then default behavior per caller)
func readString(s types.String, env string) string {
	if !s.IsNull() && !s.IsUnknown() {
		return s.ValueString()
	}
	if env == "" {
		return ""
	}
	return os.Getenv(env)
}

func readInt64Default(v types.Int64, def int) int {
	if !v.IsNull() && !v.IsUnknown() {
		return int(v.ValueInt64())
	}
	return def
}

func readBoolDefault(v types.Bool, def bool) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	return def
}

// readStringWithAliases reads a string preferring the HCL value, then a canonical env var,
// then any number of alias env vars in order.
func readStringWithAliases(s types.String, canonical string, aliases ...string) string {
	// Prefer HCL, then canonical env
	if v := readString(s, canonical); v != "" {
		return v
	}
	// Fallback to aliases in provided order
	for _, a := range aliases {
		if a == "" {
			continue
		}
		if v := os.Getenv(a); v != "" {
			return v
		}
	}
	return ""
}

// configuration derivation (unified) to avoid duplicated parsing across sections
func deriveResolvedConfig(data JiraProviderModel) resolvedConfig {
	// Base
	authMethod := readString(data.AuthMethod, "")
	if authMethod == "" {
		authMethod = defaultAuthMethod
	}
	endpoint := readStringWithAliases(data.Endpoint, "JIRA_ENDPOINT", "JIRA_BASE_URL")

	// Auth
	email := readStringWithAliases(data.APIAuthEmail, "JIRA_API_EMAIL", "JIRA_EMAIL")
	apiToken := readString(data.APIToken, "JIRA_API_TOKEN")
	username := readString(data.Username, "JIRA_USERNAME")
	password := readString(data.Password, "JIRA_PASSWORD")

	// HTTP
	httpTimeoutSeconds := readInt64Default(data.HTTPTimeoutSeconds, defaultHTTPTimeoutSeconds)

	// Retry
	retryOn4295xx := readBoolDefault(data.RetryOn4295xx, defaultRetryOn4295xx)
	retryMaxAttempts := readInt64Default(data.RetryMaxAttempts, defaultRetryMaxAttempts)
	retryInitialBackoffMs := readInt64Default(data.RetryInitialBackoffMs, defaultRetryInitialBackoffMs)
	retryMaxBackoffMs := readInt64Default(data.RetryMaxBackoffMs, defaultRetryMaxBackoffMs)

	// Privacy & Redaction
	mode := readString(data.EmailRedactionMode, "JIRA_EMAIL_REDACTION_MODE")
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" || (mode != "full" && mode != "mask") {
		mode = defaultEmailRedactionMode
	}

	return resolvedConfig{
		endpoint:              endpoint,
		authMethod:            authMethod,
		email:                 email,
		apiToken:              apiToken,
		username:              username,
		password:              password,
		httpTimeoutSeconds:    httpTimeoutSeconds,
		retryOn4295xx:         retryOn4295xx,
		retryMaxAttempts:      retryMaxAttempts,
		retryInitialBackoffMs: retryInitialBackoffMs,
		retryMaxBackoffMs:     retryMaxBackoffMs,
		emailRedactionMode:    mode,
	}
}

// validation per-section
func validateBase(rc resolvedConfig) []validationErr {
	var errs []validationErr
	if rc.endpoint == "" {
		errs = append(errs, validationErr{attr: attrEndpoint, summary: "Missing Endpoint Configuration.", detail: "Provide 'endpoint' or set JIRA_ENDPOINT (or JIRA_BASE_URL alias) environment variable."})
	}
	if rc.authMethod != "api_token" && rc.authMethod != "basic" {
		errs = append(errs, validationErr{attr: attrAuthMethod, summary: "Invalid Auth Method Configuration.", detail: "auth_method must be 'api_token' or 'basic'."})
	}
	return errs
}

func validateHTTP(rc resolvedConfig) []validationErr {
	if rc.httpTimeoutSeconds < 1 || rc.httpTimeoutSeconds > 600 {
		return []validationErr{{attr: attrHTTPTimeoutSeconds, summary: "Invalid HTTP Timeout Configuration.", detail: fmt.Sprintf("http_timeout_seconds must be between 1 and 600 seconds; got %d", rc.httpTimeoutSeconds)}}
	}
	return nil
}

func validateRetry(rc resolvedConfig) []validationErr {
	if !rc.retryOn4295xx {
		return nil
	}
	var errs []validationErr
	if rc.retryMaxAttempts < 1 || rc.retryMaxAttempts > 10 {
		errs = append(errs, validationErr{attr: attrRetryMaxAttempts, summary: "Invalid Retry Attempts Configuration.", detail: fmt.Sprintf("retry_max_attempts must be between 1 and 10; got %d", rc.retryMaxAttempts)})
	}
	if rc.retryInitialBackoffMs < 100 || rc.retryInitialBackoffMs > 600000 {
		errs = append(errs, validationErr{attr: attrRetryInitialBackoff, summary: "Invalid Retry Backoff Configuration.", detail: fmt.Sprintf("retry_initial_backoff_ms must be between 100 and 600000 milliseconds; got %d", rc.retryInitialBackoffMs)})
	}
	if rc.retryMaxBackoffMs < 100 || rc.retryMaxBackoffMs > 600000 {
		errs = append(errs, validationErr{attr: attrRetryMaxBackoff, summary: "Invalid Retry Backoff Configuration.", detail: fmt.Sprintf("retry_max_backoff_ms must be between 100 and 600000 milliseconds; got %d", rc.retryMaxBackoffMs)})
	}
	if rc.retryInitialBackoffMs > rc.retryMaxBackoffMs {
		errs = append(errs, validationErr{attr: attrRetryInitialBackoff, summary: "Invalid Retry Backoff Configuration.", detail: "retry_initial_backoff_ms must be less than or equal to retry_max_backoff_ms."})
	}
	return errs
}

func validateAuth(rc resolvedConfig) []validationErr {
	var errs []validationErr
	if rc.email != "" && rc.username != "" {
		return []validationErr{
			{attr: attrAPIAuthEmail, summary: "Conflicting credentials.", detail: "api_auth_email conflicts with username. Choose API token (api_auth_email + api_token) or basic (username + password), not both."},
			{attr: attrUsername, summary: "Conflicting credentials.", detail: "username conflicts with api_auth_email. Choose API token (api_auth_email + api_token) or basic (username + password), not both."},
		}
	}
	if rc.email == "" && rc.username == "" {
		return []validationErr{
			{attr: attrAPIAuthEmail, summary: "Missing credentials.", detail: "Provide api_auth_email with api_token for API token authentication, or set auth_method = \"basic\" and use username + password."},
			{attr: attrUsername, summary: "Missing credentials.", detail: "Provide username with password for basic authentication, or use api_auth_email + api_token for API token auth."},
		}
	}

	switch rc.authMethod {
	case "api_token":
		if rc.email == "" {
			errs = append(errs, validationErr{attr: attrAPIAuthEmail, summary: "Missing API Auth Email Configuration.", detail: "Provide 'api_auth_email' or set JIRA_API_EMAIL."})
		}
		if rc.apiToken == "" {
			errs = append(errs, validationErr{attr: attrAPIToken, summary: "Missing API Token Configuration.", detail: "Provide 'api_token' or set JIRA_API_TOKEN."})
		}
		if rc.username != "" {
			errs = append(errs, validationErr{attr: attrUsername, summary: "Attribute not allowed with api_token auth_method.", detail: "Remove 'username' (and 'password') or set auth_method = \"basic\"."})
		}
		if rc.password != "" {
			errs = append(errs, validationErr{attr: attrPassword, summary: "Attribute not allowed with api_token auth_method.", detail: "Remove 'password' (and 'username') or set auth_method = \"basic\"."})
		}
	case "basic":
		if rc.username == "" {
			errs = append(errs, validationErr{attr: attrUsername, summary: "Missing Username Configuration.", detail: "Provide 'username' or set JIRA_USERNAME."})
		}
		if rc.password == "" {
			errs = append(errs, validationErr{attr: attrPassword, summary: "Missing Password Configuration.", detail: "Provide 'password' or set JIRA_PASSWORD."})
		}
		if rc.email != "" {
			errs = append(errs, validationErr{attr: attrAPIAuthEmail, summary: "Attribute not allowed with basic auth_method.", detail: "Remove 'api_auth_email' (and 'api_token') or set auth_method = \"api_token\"."})
		}
		if rc.apiToken != "" {
			errs = append(errs, validationErr{attr: attrAPIToken, summary: "Attribute not allowed with basic auth_method.", detail: "Remove 'api_token' (and 'api_auth_email') or set auth_method = \"api_token\"."})
		}
	default:
		// already handled in validateBase, keep for completeness if validateBase is skipped
		errs = append(errs, validationErr{attr: attrAuthMethod, summary: "Invalid Auth Method Configuration.", detail: "auth_method must be 'api_token' or 'basic'."})
	}
	return errs
}

func validateResolvedConfig(rc resolvedConfig) []validationErr {
	var all []validationErr
	all = append(all, validateBase(rc)...)
	if len(all) == 0 { // if base fails, skip noisy follow-ups
		all = append(all, validateHTTP(rc)...)
		all = append(all, validateRetry(rc)...)
		all = append(all, validateAuth(rc)...)
	}

	// Collect validationErrors during checks above
	// errs := []validationError{ ... }

	// Before returning, sanitize any secrets from messages to prevent leakage.
	for i := range all {
		all[i] = sanitizeValidationError(all[i], rc)
	}
	return all
}

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

// buildHTTPClient constructs the HTTP client with optional retry/backoff policy.
func buildHTTPClient(rc resolvedConfig) *http.Client {
	if rc.retryOn4295xx {
		rcClient := retryablehttp.NewClient()
		rcClient.RetryMax = rc.retryMaxAttempts
		rcClient.RetryWaitMin = time.Duration(rc.retryInitialBackoffMs) * time.Millisecond
		rcClient.RetryWaitMax = time.Duration(rc.retryMaxBackoffMs) * time.Millisecond
		// keep default CheckRetry (retries on 429/5xx and honors Retry-After)
		// disable noisy logging unless debug is desired
		rcClient.Logger = nil
		httpClient := rcClient.StandardClient()
		httpClient.Timeout = time.Duration(rc.httpTimeoutSeconds) * time.Second
		return httpClient
	}
	return &http.Client{Timeout: time.Duration(rc.httpTimeoutSeconds) * time.Second}
}

// initJiraClient creates the Jira client, sets authentication and user agent.
func (j *JiraProvider) initJiraClient(httpClient *http.Client, rc resolvedConfig) (*jira.Client, error) {
	client, err := jira.New(httpClient, rc.endpoint)
	if err != nil {
		return nil, err
	}

	switch rc.authMethod {
	case "api_token":
		client.Auth.SetBasicAuth(rc.email, rc.apiToken)
	case "basic":
		client.Auth.SetBasicAuth(rc.username, rc.password)
	default:
		// Should be validated earlier; return an explicit error if reached.
		return nil, fmt.Errorf("invalid auth_method %q", rc.authMethod)
	}

	client.Auth.SetUserAgent(fmt.Sprintf("devops-wiz/terraform-provider-jira/%s", j.version))
	return client, nil
}

// testConnection checks API connectivity and appends diagnostics on failure.
func (j *JiraProvider) testConnection(ctx context.Context, client *jira.Client, diags *diag.Diagnostics) bool {
	_, apiResp, err := client.MySelf.Details(ctx, nil)
	return EnsureSuccessOrDiagFromScheme(ctx, "authenticate (myself)", apiResp, err, diags)
}
