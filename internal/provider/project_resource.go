// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	baseJira
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

// Wrapper functions to adapt go-atlassian signatures to generic helper expectations.
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

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Create)
	defer cancel()
	CreateResource(ctx, req, resp, &projectResourceModel{}, r.createProject)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()
	ReadResource(ctx, req, resp, &projectResourceModel{}, r.getProject)
}

// updateProject adapts the generic UpdateResource helper to Jira's update payload by
// converting a ProjectPayloadScheme (from the model) into a ProjectUpdateScheme,
// performing the update, and then reading back the full project for state mapping.
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
	UpdateResource(ctx, req, resp, &projectResourceModel{}, r.updateProject)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Delete)
	defer cancel()
	DeleteResource(ctx, req, resp, &projectResourceModel{}, r.deleteProject)
}

func (r *projectResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()
	ImportResource(ctx, request, response, &projectResourceModel{}, r.getProject)
}
