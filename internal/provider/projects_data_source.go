// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*projectsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*projectsDataSource)(nil)

// NewProjectsDataSource returns the Terraform data source implementation for jira_projects (list/filter; pagination).
func NewProjectsDataSource() datasource.DataSource { return &projectsDataSource{} }

type projectsDataSource struct {
	baseJira
}

type projectsDataSourceModel struct {
	// Optional filters
	Ids      types.List   `tfsdk:"ids"`
	Keys     types.List   `tfsdk:"keys"`
	TypeKeys types.List   `tfsdk:"type_keys"`
	Query    types.String `tfsdk:"query"`
	OrderBy  types.String `tfsdk:"order_by"`

	// Outputs
	Projects types.Map `tfsdk:"projects"`
}

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Jira projects with optional filters by IDs, keys, or names. Results are deterministically ordered by project key.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by project IDs (string).",
			},
			"keys": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by project keys.",
			},
			"type_keys": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filter by project type keys.",
			},
			"query": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter the results using a literal string. Projects with a matching key or name are returned (case insensitive).",
			},
			"order_by": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "",
			},
			"projects": schema.MapAttribute{
				Computed: true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"id":               types.StringType,
					"key":              types.StringType,
					"name":             types.StringType,
					"project_type_key": types.StringType,
					"description":      types.StringType,
					"url":              types.StringType,
					"assignee_type":    types.StringType,
					"lead_account_id":  types.StringType,
					"category_id":      types.Int64Type,
				}},
				MarkdownDescription: "Map of projects keyed by project ID. Values include key, name, project_type_key, description, url, assignee_type, lead_account_id, and category_id.",
			},
		},
	}
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, d.providerTimeouts.Read)
	defer cancel()

	var data projectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idsStrs, deferIDs := getKnownStrings(ctx, data.Ids, "ids", &resp.Diagnostics)
	if resp.Diagnostics.HasError() || deferIDs {
		return
	}
	var ids []int
	if len(idsStrs) > 0 {
		ids = make([]int, len(idsStrs))
		for i, s := range idsStrs {
			if s == "" {
				resp.Diagnostics.AddAttributeError(path.Root("ids"), "Invalid ids list", "IDs must be non-empty strings representing numeric project IDs.")
				return
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("ids"), "Invalid ids list", fmt.Sprintf("Element at index %d is not a valid numeric project ID: %q", i, s))
				return
			}
			ids[i] = v
		}
	}

	keys, deferKeys := getKnownStrings(ctx, data.Keys, "keys", &resp.Diagnostics)
	if resp.Diagnostics.HasError() || deferKeys {
		return
	}

	typeKeys, deferTypeKeys := getKnownStrings(ctx, data.TypeKeys, "type_keys", &resp.Diagnostics)
	if resp.Diagnostics.HasError() || deferTypeKeys {
		return
	}

	// Fetch all projects via paginated API. Prefer server pagination where available.
	var all []*models.ProjectScheme

	opts := &models.ProjectSearchOptionsScheme{
		IDs:      ids,
		Keys:     keys,
		TypeKeys: typeKeys,
	}
	if !data.OrderBy.IsNull() {
		opts.OrderBy = data.OrderBy.ValueString()
	}
	if !data.Query.IsNull() {
		opts.Query = data.Query.ValueString()
	}

	startAt := 0
	maxResults := 50
	for {
		searchResults, apiResp, err := d.client.Project.Search(ctx, opts, startAt, maxResults)
		if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "list projects", apiResp, err, &resp.Diagnostics, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
			return
		}

		values := searchResults.Values
		if len(values) == 0 {
			break
		}
		all = append(all, values...)

		if searchResults.IsLast {
			break
		}
		// Advance by observed page size to avoid duplicates if server returns partial pages
		startAt += len(values)
	}

	// Deterministic ordering: by project key, then by ID for stable state
	sort.Slice(all, func(i, j int) bool {
		if all[i].Key == all[j].Key {
			return all[i].ID < all[j].ID
		}
		return all[i].Key < all[j].Key
	})

	// Map to state objects keyed by canonical ID
	objMap := make(map[string]projectResourceModel, len(all))
	for _, p := range all {
		var m projectResourceModel
		if diags := m.TransformToState(ctx, p); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		objMap[m.ID.ValueString()] = m
	}

	var mDiag diag.Diagnostics
	data.Projects, mDiag = types.MapValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":               types.StringType,
		"key":              types.StringType,
		"name":             types.StringType,
		"project_type_key": types.StringType,
		"description":      types.StringType,
		"url":              types.StringType,
		"assignee_type":    types.StringType,
		"lead_account_id":  types.StringType,
		"category_id":      types.Int64Type,
	}}, objMap)
	if mDiag.HasError() {
		resp.Diagnostics.AddAttributeError(
			path.Root("projects"),
			"Failed to build projects map",
			fmt.Sprintf("Could not encode %d projects into state. See diagnostics for details.", len(objMap)),
		)
		resp.Diagnostics.Append(mDiag...)
		return
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
