// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/ctreminiom/go-atlassian/v2/service/jira"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var _ resource.Resource = (*projectCategoryResource)(nil)
var _ resource.ResourceWithConfigure = (*projectCategoryResource)(nil)
var _ resource.ResourceWithImportState = (*projectCategoryResource)(nil)
var _ resource.ResourceWithValidateConfig = (*projectCategoryResource)(nil)

// NewProjectCategoryResource returns the Terraform resource implementation for jira_project_category.
func NewProjectCategoryResource() resource.Resource { return &projectCategoryResource{} }

type projectCategoryResource struct {
	ServiceClient
	categoryService jira.ProjectCategoryConnector
}

func (r *projectCategoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_category"
}

func (r *projectCategoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*JiraProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected JiraProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = provider.client
	r.categoryService = provider.client.Project.Category
	r.providerTimeouts = provider.providerTimeouts
}

func (r *projectCategoryResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data projectCategoryResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Name.IsNull() || data.Name.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Missing required attribute", "The 'name' attribute is required.")
	}
}

func (r *projectCategoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jira project category.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The unique identifier of the project category (string ID).",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the project category.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the project category.",
			},
		},
	}
}

// Wrapper functions to adapt go-atlassian signatures for this resource.
func (r *projectCategoryResource) createCategory(ctx context.Context, p *models.ProjectCategoryPayloadScheme) (*models.ProjectCategoryScheme, *models.ResponseScheme, error) {
	return r.categoryService.Create(ctx, p)
}

func (r *projectCategoryResource) getCategory(ctx context.Context, id string) (*models.ProjectCategoryScheme, *models.ResponseScheme, error) {
	// API expects integer ID
	i, err := strconv.Atoi(id)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid category id %q: %w", id, err)
	}
	return r.categoryService.Get(ctx, i)
}

func (r *projectCategoryResource) updateCategory(ctx context.Context, id string, p *models.ProjectCategoryPayloadScheme) (*models.ProjectCategoryScheme, *models.ResponseScheme, error) {
	i, err := strconv.Atoi(id)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid category id %q: %w", id, err)
	}
	return r.categoryService.Update(ctx, i, p)
}

func (r *projectCategoryResource) deleteCategory(ctx context.Context, id string) (*models.ResponseScheme, error) {
	i, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid category id %q: %w", id, err)
	}
	return r.categoryService.Delete(ctx, i)
}

func (r *projectCategoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Create)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoCreate(
		ctx,
		func(ctx context.Context, dst *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.Plan.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(resp.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCategoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoRead(
		ctx,
		func(ctx context.Context, dst *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.State.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(resp.State.Set(ctx, src)...)
			return d
		},
		func(ctx context.Context) { resp.State.RemoveResource(ctx) },
		ensureWith(&resp.Diagnostics),
		HTTPStatusFromScheme,
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCategoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Update)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoUpdate(
		ctx,
		func(ctx context.Context, dst *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.Plan.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(resp.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCategoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Delete)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoDelete(
		ctx,
		func(ctx context.Context, dst *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.State.Get(ctx, dst)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectCategoryResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	diags := DoImport[projectCategoryResourceModel, *models.ProjectCategoryScheme](
		ctx,
		request.ID,
		r.getCategory,
		func(ctx context.Context, api *models.ProjectCategoryScheme, st *projectCategoryResourceModel) diag.Diagnostics {
			return r.hooks().MapToState(ctx, api, st)
		},
		func(ctx context.Context, src *projectCategoryResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(response.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&response.Diagnostics),
	)
	response.Diagnostics.Append(diags...)
}

// hooks returns the CRUD hooks for the generic runner.
func (r *projectCategoryResource) hooks() CRUDHooks[projectCategoryResourceModel, *models.ProjectCategoryPayloadScheme, *models.ProjectCategoryScheme] {
	return CRUDHooks[projectCategoryResourceModel, *models.ProjectCategoryPayloadScheme, *models.ProjectCategoryScheme]{
		BuildPayload: func(ctx context.Context, st *projectCategoryResourceModel) (*models.ProjectCategoryPayloadScheme, diag.Diagnostics) {
			var diags diag.Diagnostics
			p := &models.ProjectCategoryPayloadScheme{
				Name:        st.Name.ValueString(),
				Description: st.Description.ValueString(),
			}
			return p, diags
		},
		APICreate:               r.createCategory,
		APIRead:                 r.getCategory,
		APIUpdate:               r.updateCategory,
		APIDelete:               r.deleteCategory,
		ExtractID:               func(st *projectCategoryResourceModel) string { return st.ID.ValueString() },
		MapToState:              mapProjectCategorySchemeToModel,
		TreatDelete404AsSuccess: true,
	}
}
