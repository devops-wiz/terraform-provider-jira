// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
	"regexp"
)

var endpoint = os.Getenv("JIRA_ENDPOINT")
var email = os.Getenv("JIRA_API_EMAIL")
var apiToken = os.Getenv("JIRA_API_TOKEN")
var username = os.Getenv("JIRA_USERNAME")
var password = os.Getenv("JIRA_PASSWORD")

// Ensure JiraProvider satisfies various provider interfaces.
var _ provider.Provider = &JiraProvider{}
var _ provider.ProviderWithValidateConfig = &JiraProvider{}

// JiraProvider defines the provider implementation.
type JiraProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string

	// client represents an instance of the Jira client used to interact with the Jira API.
	client jira.Client
}

// JiraProviderModel describes the provider data model.
type JiraProviderModel struct {
	// Base Configuration
	Endpoint types.String `tfsdk:"endpoint"`

	AuthMethod types.String `tfsdk:"auth_method"`

	// API Token Authentication
	APIToken     types.String `tfsdk:"api_token"`
	APIAuthEmail types.String `tfsdk:"api_auth_email"`

	// Basic Authentication
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (j *JiraProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jira"
	resp.Version = j.version
}

func (j *JiraProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Jira provider for interacting with Jira instances using the go-jira library.",
		Attributes: map[string]schema.Attribute{
			// Base Configuration
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base Endpoint of the Jira instance (e.g., 'https://your-domain.atlassian.net').",
				Required:            true,
			},

			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method to use for Jira. Defaults to API token authentication. Accepts values `api_token` or `basic`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`(api_token|basic|^$)`), "auth_method must be one of 'api_token' or 'basic'."),
				},
			},

			// API Token Authentication (Recommended for Jira Cloud)
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token (PAT) for authentication. Required when using API token authentication with email.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_auth_email": schema.StringAttribute{
				MarkdownDescription: "Email address associated with the API token. Required when using API token authentication.",
				Optional:            true,
			},

			// Basic Authentication (For self-hosted Jira)
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for basic authentication. Required when using basic authentication with password.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for basic authentication. Required when using basic authentication.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (j *JiraProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data JiraProviderModel

	switch {
	case email != "" && username != "":
		resp.Diagnostics.AddError("Conflicting environment variables set.", "environment variables JIRA_USERNAME and JIRA_API_EMAIL cannot be set at the same time, as they are used for separate authentication methods.")
		return
	case email != "" && apiToken == "":
		resp.Diagnostics.AddError("Missing environment variables.", "environment variable JIRA_API_EMAIL requires environment variable JIRA_API_TOKEN.")
		return
	case username != "" && password == "":
		resp.Diagnostics.AddError("Missing environment variables.", "environment variable JIRA_USERNAME requires environment variable JIRA_PASSWORD.")
		return
	case (email != "" && apiToken != "") || (username != "" && password != ""):
		return
	}

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case !data.APIAuthEmail.IsNull() && !data.Username.IsNull():
		resp.Diagnostics.AddError("Conflicting provider attributes set.", "username and email cannot be set at the same time, as they are used for separate authentication methods.")
		return
	case data.APIAuthEmail.IsNull() && data.Username.IsNull():
		resp.Diagnostics.AddError("Missing provider attributes.", "username or email must be set to authenticate with Jira.")
		return
	case !data.APIAuthEmail.IsNull() && data.APIToken.IsNull():
		resp.Diagnostics.AddError("Missing provider attributes.", "api_token must be set when using API token authentication.")
		return
	case !data.Username.IsNull() && data.Password.IsNull():
		resp.Diagnostics.AddError("Missing provider attributes.", "password must be set when using basic authentication.")
		return
	}

}

func (j *JiraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	var data JiraProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing Endpoint Configuration.", "While configuring the provider, "+
			"the Jira endpoint was not found in "+
			"the JIRA_ENDPOINT environment variable. or provider "+
			"configuration block 'endpoint' attribute.")
		return
	}

	if data.AuthMethod.IsNull() {
		data.AuthMethod = types.StringValue("api_token")
	}

	var tp jira.BasicAuthTransport

	switch data.AuthMethod.ValueString() {
	case "api_token":
		if !data.APIAuthEmail.IsNull() {
			email = data.APIAuthEmail.ValueString()
		}
		if !data.APIToken.IsNull() {
			apiToken = data.APIToken.ValueString()
		}

		if email == "" {
			resp.Diagnostics.AddAttributeError(path.Root("api_auth_email"), "Missing API Auth Email Configuration.", "While configuring the provider, "+
				"the Jira API email was not found in "+
				"the JIRA_API_EMAIL environment variable. or provider "+
				"configuration block 'api_auth_email' attribute.")
			return
		}

		if apiToken == "" {
			resp.Diagnostics.AddAttributeError(path.Root("api_token"), "Missing API Token Configuration.", "While configuring the provider, "+
				"the Jira API token was not found in "+
				"the JIRA_API_TOKEN environment variable. or provider "+
				"configuration block 'api_token' attribute.")
			return
		}

		tp = jira.BasicAuthTransport{
			Username: email,
			Password: apiToken,
		}
	case "basic":
		if !data.Username.IsNull() {
			username = data.Username.ValueString()
		}
		if !data.Password.IsNull() {
			password = data.Password.ValueString()
		}

		if username == "" {
			resp.Diagnostics.AddAttributeError(path.Root("username"), "Missing Username Configuration.", "While configuring the provider, "+
				"the Jira username was not found in "+
				"the JIRA_USERNAME environment variable. or provider "+
				"configuration block 'username' attribute.")
			return
		}
		if password == "" {
			resp.Diagnostics.AddAttributeError(path.Root("password"), "Missing Password Configuration.", "While configuring the provider, "+
				"the Jira password was not found in "+
				"the JIRA_PASSWORD environment variable. or provider "+
				"configuration block 'password' attribute.")
			return
		}

		tp = jira.BasicAuthTransport{
			Username: username,
			Password: password,
		}
	default:
		resp.Diagnostics.AddError("Invalid Auth Method Configuration.", "While configuring the provider, "+
			"the auth_method was not found in "+
			"the provider configuration block 'auth_method' attribute. "+
			"Valid values are 'api_token' or 'basic'.")
		return
	}

	// Create the client
	client, err := jira.NewClient(tp.Client(), endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jira client", err.Error())
		return
	}

	// Store the client
	j.client = *client

	// Test the connection
	_, _, err = client.User.GetSelf()
	if err != nil {
		resp.Diagnostics.AddError("Error connecting to Jira",
			"Failed to authenticate with Jira: "+err.Error())
		return
	}

}

func (j *JiraProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (j *JiraProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &JiraProvider{
			version: version,
		}
	}
}
