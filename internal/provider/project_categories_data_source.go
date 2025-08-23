// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/service/jira"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*projectCategoriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*projectCategoriesDataSource)(nil)

// NewProjectCategoriesDataSource returns the Terraform data source implementation for jira_project_categories.
func NewProjectCategoriesDataSource() datasource.DataSource { return &projectCategoriesDataSource{} }

type projectCategoriesDataSource struct {
	baseJira
	categoryService jira.ProjectCategoryConnector
}

type projectCategoriesDataSourceModel struct {
	Ids        types.List `tfsdk:"ids"`
	Names      types.List `tfsdk:"names"`
	Categories types.Map  `tfsdk:"categories"`
}

func (d *projectCategoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_categories"
}

func (d *projectCategoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List Jira project categories with optional filtering by IDs or names. Results are returned as a map keyed by category ID for stability across renames.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Filter by project category IDs (string). If omitted, all categories are returned.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(path.MatchRoot("names")),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"names": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Filter by project category names (case-insensitive). If omitted, all categories are returned.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(path.MatchRoot("ids")),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"categories": schema.MapNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Map of project categories keyed by ID. Each value includes id, name, and description.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the project category (string ID).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The display name of the project category.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A description of the project category.",
						},
					},
				},
			},
		},
	}
}

func (d *projectCategoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.categoryService = provider.client.Project.Category
	d.providerTimeouts = provider.providerTimeouts
}

func (d *projectCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, d.providerTimeouts.Read)
	defer cancel()

	var data projectCategoriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, deferIDs := getKnownStrings(ctx, data.Ids, "ids", &resp.Diagnostics)
	if resp.Diagnostics.HasError() || deferIDs {
		return
	}
	names, deferNames := getKnownStrings(ctx, data.Names, "names", &resp.Diagnostics)
	if resp.Diagnostics.HasError() || deferNames {
		return
	}

	// Fetch all categories
	cats, apiResp, err := d.categoryService.Gets(ctx)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "list project categories", apiResp, err, &resp.Diagnostics, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
		return
	}

	// Deterministic order for any intermediate processing: by name (case-insensitive), then by id
	sort.SliceStable(cats, func(i, j int) bool {
		a := strings.ToLower(cats[i].Name)
		b := strings.ToLower(cats[j].Name)
		if a == b {
			return cats[i].ID < cats[j].ID
		}
		return a < b
	})

	// Build map keyed by ID, applying optional filters
	result := make(map[string]projectCategoryResourceModel)
	idFilter := map[string]struct{}{}
	for _, id := range ids {
		idFilter[id] = struct{}{}
	}
	nameFilter := map[string]struct{}{}
	for _, n := range names {
		nameFilter[strings.ToLower(n)] = struct{}{}
	}

	for _, c := range cats {
		if len(idFilter) > 0 {
			if _, ok := idFilter[c.ID]; !ok {
				continue
			}
		}
		if len(nameFilter) > 0 {
			if _, ok := nameFilter[strings.ToLower(c.Name)]; !ok {
				continue
			}
		}
		m := projectCategoryResourceModel{}
		// Reuse resource transformer for consistent mapping
		if diags := m.TransformToState(ctx, c); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		result[m.ID.ValueString()] = m
	}

	var diags diag.Diagnostics
	data.Categories, diags = types.MapValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
	}}, result)
	if diags.HasError() {
		resp.Diagnostics.AddAttributeError(
			path.Root("categories"),
			"Failed to build categories map",
			fmt.Sprintf("Could not encode %d categories into state. See diagnostics for details.", len(result)),
		)
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
