// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/ctreminiom/go-atlassian/v2/service/jira"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/constants"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*fieldDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*fieldDataSource)(nil)

// NewFieldDataSource returns the Terraform data source implementation for jira_field (lookup by ID/name).
func NewFieldDataSource() datasource.DataSource { return &fieldDataSource{} }

type fieldDataSource struct {
	ServiceClient
	fieldService jira.FieldConnector
}

type fieldDataSourceModel struct {
	// Inputs (exactly one must be provided)
	LookupID   types.String `tfsdk:"id"`
	LookupName types.String `tfsdk:"name"`

	// Outputs (all computed)
	FieldID     types.String `tfsdk:"field_id"`
	FieldName   types.String `tfsdk:"field_name"`
	Description types.String `tfsdk:"description"`
	FieldType   types.String `tfsdk:"field_type"`
}

func (d *fieldDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_field"
}

func (d *fieldDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lookup a single Jira custom field by ID or name.",
		Attributes: map[string]schema.Attribute{
			// Inputs
			"id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Field identifier (e.g., customfield_10001). Exactly one of id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Field display name. Exactly one of id or name must be set.",
			},

			// Outputs (prefixed to avoid input/output name collision in state)
			"field_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the field (e.g., customfield_10001).",
			},
			"field_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name of the field.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A description of the field.",
			},
			"field_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The field type key (short form).",
			},
		},
	}
}

func (d *fieldDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.fieldService = provider.client.Issue.Field
	d.providerTimeouts = provider.providerTimeouts
}

func (d *fieldDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, d.providerTimeouts.Read)
	defer cancel()

	var data fieldDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !data.LookupID.IsNull() && !data.LookupID.IsUnknown() && data.LookupID.ValueString() != ""
	nameSet := !data.LookupName.IsNull() && !data.LookupName.IsUnknown() && data.LookupName.ValueString() != ""

	// Exactly one of id or name must be provided
	if (idSet && nameSet) || (!idSet && !nameSet) {
		if idSet && nameSet {
			resp.Diagnostics.AddError(
				"Invalid configuration for jira_field data source",
				"Exactly one of 'id' or 'name' must be set, but both were provided.",
			)
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Conflicts with name", "Remove either 'id' or 'name' so that only one is provided.")
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Conflicts with id", "Remove either 'id' or 'name' so that only one is provided.")
		} else {
			resp.Diagnostics.AddError(
				"Missing configuration for jira_field data source",
				"Exactly one of 'id' or 'name' must be set to lookup a field.",
			)
			resp.Diagnostics.AddAttributeError(path.Root("id"), "One of id or name required", "Set 'id' (field identifier like customfield_10001) or 'name' (field display name).")
			resp.Diagnostics.AddAttributeError(path.Root("name"), "One of id or name required", "Set 'name' (field display name) or 'id' (field identifier like customfield_10001).")
		}
		return
	}

	// Fetch all fields from Jira API
	allFields, apiResp, err := d.fieldService.Gets(ctx)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "get fields", apiResp, err, &resp.Diagnostics, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
		return
	}

	// Find the matching field
	var targetField *struct {
		ID          string
		Name        string
		Description string
		Custom      string
	}

	lookupValue := data.LookupID.ValueString()
	lookupByID := idSet
	if !idSet {
		lookupValue = data.LookupName.ValueString()
		lookupByID = false
	}

	for _, field := range allFields {
		if field == nil {
			continue
		}
		
		if lookupByID {
			if field.ID == lookupValue {
				targetField = &struct {
					ID          string
					Name        string
					Description string
					Custom      string
				}{
					ID:          field.ID,
					Name:        field.Name,
					Description: field.Description,
					Custom:      field.Schema.Custom,
				}
				break
			}
		} else {
			if field.Name == lookupValue {
				targetField = &struct {
					ID          string
					Name        string
					Description string
					Custom      string
				}{
					ID:          field.ID,
					Name:        field.Name,
					Description: field.Description,
					Custom:      field.Schema.Custom,
				}
				break
			}
		}
	}

	if targetField == nil {
		if lookupByID {
			resp.Diagnostics.AddError(
				"Field not found",
				fmt.Sprintf("No field found with ID '%s'", lookupValue),
			)
		} else {
			resp.Diagnostics.AddError(
				"Field not found",
				fmt.Sprintf("No field found with name '%s'", lookupValue),
			)
		}
		return
	}

	// Map API model to data source outputs
	data.FieldID = types.StringValue(targetField.ID)
	data.FieldName = types.StringValue(targetField.Name)
	data.FieldType = types.StringValue(constants.GetFieldTypeShort(targetField.Custom))

	if targetField.Description != "" {
		data.Description = types.StringValue(targetField.Description)
	} else {
		data.Description = types.StringNull()
	}

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.AddError(
			"Failed to set data source state",
			"An unexpected error occurred while writing computed data to Terraform state. See diagnostics for details.",
		)
		resp.Diagnostics.Append(diags...)
		return
	}
}