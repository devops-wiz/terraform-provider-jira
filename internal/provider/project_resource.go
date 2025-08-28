// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ resource.Resource = (*projectResource)(nil)
var _ resource.ResourceWithConfigure = (*projectResource)(nil)
var _ resource.ResourceWithImportState = (*projectResource)(nil)

// NewProjectResource returns the Terraform resource implementation for jira_project.
func NewProjectResource() resource.Resource { return &projectResource{} }

type projectResource struct {
	ServiceClient
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.providerTimeouts = provider.providerTimeouts
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jira project. Projects group issues and configuration within Jira Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The unique identifier of the project (string ID).",
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The project key (e.g., ABC).",
				Validators:          []validator.String{stringvalidator.LengthBetween(2, 10)},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the project.",
			},
			"project_type_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The project type key (e.g., software, service_desk, business).",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Project description.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project URL (info link). This value is managed by Jira and is read-only in Terraform.",
			},
			"assignee_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Default assignee type (e.g., PROJECT_LEAD or UNASSIGNED).",
			},
			"lead_account_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Account ID for the project lead.",
			},
			"category_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				MarkdownDescription: "Project category ID.",
			},
		},
	}
}

// Wrapper functions to adapt go-atlassian signatures for this resource.
func (r *projectResource) createProject(ctx context.Context, p *models.ProjectPayloadScheme) (*models.ProjectScheme, *models.ResponseScheme, error) {
	created, rs, err := r.client.Project.Create(ctx, p)
	if err != nil || created == nil {
		return nil, rs, err
	}
	id := ""
	if created.ID != 0 {
		id = strconv.Itoa(created.ID)
	} else if p != nil && p.Key != "" {
		id = p.Key
	}
	proj, rs2, err2 := r.client.Project.Get(ctx, id, nil)
	if err2 != nil {
		return nil, rs2, err2
	}
	return proj, rs2, nil
}

func (r *projectResource) getProject(ctx context.Context, id string) (*models.ProjectScheme, *models.ResponseScheme, error) {
	return r.client.Project.Get(ctx, id, nil)
}

func (r *projectResource) deleteProject(ctx context.Context, id string) (*models.ResponseScheme, error) {
	return r.client.Project.Delete(ctx, id, false)
}

// hooks returns the CRUD hooks for the generic runner.
func (r *projectResource) hooks() CRUDHooks[projectResourceModel, models.ProjectPayloadScheme, *models.ProjectScheme] {
	return CRUDHooks[projectResourceModel, models.ProjectPayloadScheme, *models.ProjectScheme]{
		BuildPayload: func(ctx context.Context, st *projectResourceModel) (*models.ProjectPayloadScheme, diag.Diagnostics) {
			var diags diag.Diagnostics
			p := &models.ProjectPayloadScheme{
				Key:            st.Key.ValueString(),
				Name:           st.Name.ValueString(),
				ProjectTypeKey: st.ProjectTypeKey.ValueString(),
				Description:    st.Description.ValueString(),
				AssigneeType:   st.AssigneeType.ValueString(),
				LeadAccountID:  st.LeadAccountID.ValueString(),
			}
			if !st.CategoryID.IsNull() && !st.CategoryID.IsUnknown() {
				p.CategoryID = int(st.CategoryID.ValueInt64())
			}
			return p, diags
		},
		APICreate:               r.createProject, // already does Create→Get
		APIRead:                 r.getProject,
		APIUpdate:               r.updateProject, // already does Update→Get
		APIDelete:               r.deleteProject,
		ExtractID:               func(st *projectResourceModel) string { return st.ID.ValueString() },
		MapToState:              mapProjectSchemeToModel,
		TreatDelete404AsSuccess: true,
	}
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Create)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoCreate(
		ctx,
		func(ctx context.Context, dst *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.Plan.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(resp.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoRead(
		ctx,
		func(ctx context.Context, dst *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.State.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectResourceModel) diag.Diagnostics {
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

// updateProject converts a ProjectPayloadScheme into a ProjectUpdateScheme,
// performs the update, and then reads back the full project for state mapping.
func (r *projectResource) updateProject(ctx context.Context, id string, p *models.ProjectPayloadScheme) (*models.ProjectScheme, *models.ResponseScheme, error) {
	u := &models.ProjectUpdateScheme{
		Name:          p.Name,
		Description:   p.Description,
		AssigneeType:  p.AssigneeType,
		LeadAccountID: p.LeadAccountID,
	}
	if p.CategoryID != 0 {
		u.CategoryID = p.CategoryID
	}

	_, rs, err := r.client.Project.Update(ctx, id, u)
	if err != nil {
		return nil, rs, err
	}

	proj, rs2, err2 := r.client.Project.Get(ctx, id, nil)
	if err2 != nil {
		return nil, rs2, err2
	}
	return proj, rs2, nil
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Update)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoUpdate(
		ctx,
		func(ctx context.Context, dst *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.Plan.Get(ctx, dst)...)
			return d
		},
		func(ctx context.Context, src *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(resp.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Delete)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoDelete(
		ctx,
		func(ctx context.Context, dst *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(req.State.Get(ctx, dst)...)
			return d
		},
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	diags := DoImport[projectResourceModel, *models.ProjectScheme](
		ctx,
		request.ID,
		r.getProject,
		func(ctx context.Context, api *models.ProjectScheme, st *projectResourceModel) diag.Diagnostics {
			// Reuse the same mapper as the CRUD hooks
			return r.hooks().MapToState(ctx, api, st)
		},
		func(ctx context.Context, src *projectResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.Append(response.State.Set(ctx, src)...)
			return d
		},
		ensureWith(&response.Diagnostics),
	)
	response.Diagnostics.Append(diags...)
}
