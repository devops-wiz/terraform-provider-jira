// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*projectDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)

// NewProjectDataSource returns the Terraform data source implementation for jira_project (lookup by key/ID).
func NewProjectDataSource() datasource.DataSource { return &projectDataSource{} }

type projectDataSource struct {
	baseJira
}

type projectDataSourceModel struct {
	// Inputs (exactly one must be provided)
	LookupID  types.String `tfsdk:"id"`
	LookupKey types.String `tfsdk:"key"`

	// Outputs (all computed)
	ID             types.String `tfsdk:"project_id"`
	KeyOut         types.String `tfsdk:"project_key"`
	Name           types.String `tfsdk:"name"`
	ProjectTypeKey types.String `tfsdk:"project_type_key"`
	Description    types.String `tfsdk:"description"`
	URL            types.String `tfsdk:"url"`
	AssigneeType   types.String `tfsdk:"assignee_type"`
	LeadAccountID  types.String `tfsdk:"lead_account_id"`
	CategoryID     types.Int64  `tfsdk:"category_id"`
}

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a single Jira project by key or ID.",
		Attributes: map[string]schema.Attribute{
			// Inputs
			"id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Project identifier (string). Exactly one of id or key must be set.",
			},
			"key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Project key (e.g., ABC). Exactly one of id or key must be set.",
			},

			// Outputs (prefixed to avoid input/output name collision in state)
			"project_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the project (string ID).",
			},
			"project_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The project key.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name of the project.",
			},
			"project_type_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The project type key (e.g., software, service_desk, business).",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project description.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project URL (info link). Read-only.",
			},
			"assignee_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default assignee type.",
			},
			"lead_account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Account ID for the project lead.",
			},
			"category_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Project category ID.",
			},
		},
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*JiraProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected JiraProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = provider.client
	d.providerTimeouts = provider.providerTimeouts
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, d.providerTimeouts.Read)
	defer cancel()

	var data projectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !data.LookupID.IsNull() && !data.LookupID.IsUnknown() && data.LookupID.ValueString() != ""
	keySet := !data.LookupKey.IsNull() && !data.LookupKey.IsUnknown() && data.LookupKey.ValueString() != ""

	// Exactly one of id or key must be provided
	if (idSet && keySet) || (!idSet && !keySet) {
		if idSet && keySet {
			resp.Diagnostics.AddError(
				"Invalid configuration for jira_project data source",
				"Exactly one of 'id' or 'key' must be set, but both were provided.",
			)
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Conflicts with key", "Remove either 'id' or 'key' so that only one is provided.")
			resp.Diagnostics.AddAttributeError(path.Root("key"), "Conflicts with id", "Remove either 'id' or 'key' so that only one is provided.")
		} else {
			resp.Diagnostics.AddError(
				"Missing configuration for jira_project data source",
				"Exactly one of 'id' or 'key' must be set to lookup a project.",
			)
			resp.Diagnostics.AddAttributeError(path.Root("id"), "One of id or key required", "Set 'id' (string project ID) or 'key' (project key).")
			resp.Diagnostics.AddAttributeError(path.Root("key"), "One of id or key required", "Set 'key' (project key) or 'id' (string project ID).")
		}
		return
	}

	idOrKey := data.LookupID.ValueString()
	if !idSet {
		idOrKey = data.LookupKey.ValueString()
	}

	// Perform lookup via Jira API. The Get endpoint accepts either project ID or key.
	proj, apiResp, err := d.client.Project.Get(ctx, idOrKey, nil)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "get project", apiResp, err, &resp.Diagnostics, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
		return
	}

	// Map API model to state using existing resource mapping for consistency.
	var m projectResourceModel
	if diags := m.TransformToState(ctx, proj); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Assign to data source output fields (prefixed versions of id/key to avoid overlap with inputs)
	data.ID = m.ID
	data.KeyOut = m.Key
	data.Name = m.Name
	data.ProjectTypeKey = m.ProjectTypeKey
	data.Description = m.Description
	data.URL = m.URL
	data.AssigneeType = m.AssigneeType
	data.LeadAccountID = m.LeadAccountID
	data.CategoryID = m.CategoryID

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.AddError(
			"Failed to set data source state",
			"An unexpected error occurred while writing computed data to Terraform state. See diagnostics for details.",
		)
		resp.Diagnostics.Append(diags...)
		return
	}
}
