// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/ctreminiom/go-atlassian/v2/service/jira"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var _ resource.Resource = (*workflowStatusResource)(nil)
var _ resource.ResourceWithConfigure = (*workflowStatusResource)(nil)
var _ resource.ResourceWithImportState = (*workflowStatusResource)(nil)

// NewWorkflowStatusResource returns the Terraform resource implementation for jira_workflow_status.
func NewWorkflowStatusResource() resource.Resource {
	return &workflowStatusResource{}
}

type workflowStatusResource struct {
	baseJira
	workflowStatusService jira.WorkflowStatusConnector
}

// GetID Implement generic helper interfaces
func (m *workflowStatusResourceModel) GetID() string {
	return m.ID.ValueString()
}

func (m *workflowStatusResourceModel) GetAPIPayload(_ context.Context) (*models.WorkflowStatusPayloadScheme, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	node := &models.WorkflowStatusNodeScheme{
		ID:             m.ID.ValueString(),
		Name:           m.Name.ValueString(),
		StatusCategory: m.StatusCategory.ValueString(),
		Description:    m.Description.ValueString(),
	}
	payload := &models.WorkflowStatusPayloadScheme{
		Statuses: []*models.WorkflowStatusNodeScheme{node},
	}
	// Include scope only during creation (when ID is not yet known)
	if m.ID.IsNull() || m.ID.IsUnknown() || m.ID.ValueString() == "" {
		payload.Scope = &models.WorkflowStatusScopeScheme{Type: "GLOBAL"}
	}
	return payload, diags
}

func (m *workflowStatusResourceModel) TransformToState(_ context.Context, apiModel *workflowStatusStateView) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if apiModel == nil {
		diags.AddError("Unexpected empty API model when mapping workflow status", "The Jira API returned no status payload to map into state.")
		return diags
	}
	m.ID = types.StringValue(apiModel.ID)
	m.Description = types.StringValue(apiModel.Description)
	m.Name = types.StringValue(apiModel.Name)
	m.StatusCategory = types.StringValue(apiModel.StatusCategory)
	return diags
}

func (r *workflowStatusResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.workflowStatusService = provider.client.Workflow.Status
	r.providerTimeouts = provider.providerTimeouts
}

func (r *workflowStatusResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_status"
}

func (r *workflowStatusResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jira workflow status.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The ID of the status.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A detailed description of the status.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the status.",
			},
			"status_category": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The category of the status in Jira. Valid values: `TODO`, `IN_PROGRESS`, `DONE`. These categories affect board columns, workflow transitions, and reporting: TODO = not started/backlog; IN_PROGRESS = actively being worked; DONE = completed/resolved. See Atlassian docs: https://support.atlassian.com/jira-cloud-administration/docs/manage-statuses-resolutions-and-priorities/#Status-categories",
				Validators: []validator.String{
					stringvalidator.OneOf("TODO", "IN_PROGRESS", "DONE"),
				},
			},
		},
	}
}

func (r *workflowStatusResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Create)
	defer cancel()
	var data workflowStatusResourceModel

	createFn := func(ctx context.Context, payload *models.WorkflowStatusPayloadScheme) (*workflowStatusStateView, *models.ResponseScheme, error) {
		statuses, apiResp, err := r.workflowStatusService.Create(ctx, payload)
		if err != nil {
			return nil, apiResp, err
		}
		if len(statuses) != 1 {
			return nil, apiResp, fmt.Errorf("expected one status from create, got %d", len(statuses))
		}
		st := statuses[0]
		view := &workflowStatusStateView{ID: st.ID, Description: st.Description, Name: st.Name, StatusCategory: st.StatusCategory}
		return view, apiResp, nil
	}

	CreateResource[*models.WorkflowStatusPayloadScheme, *workflowStatusStateView](ctx, req, resp, &data, createFn)
}

func (r *workflowStatusResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()
	var data workflowStatusResourceModel

	readFn := func(ctx context.Context, id string) (*workflowStatusStateView, *models.ResponseScheme, error) {
		statuses, apiResp, err := r.workflowStatusService.Gets(ctx, []string{id}, nil)
		if err != nil {
			return nil, apiResp, err
		}
		if len(statuses) != 1 {
			return nil, apiResp, fmt.Errorf("expected one status from read, got %d", len(statuses))
		}
		st := statuses[0]
		view := &workflowStatusStateView{ID: st.ID, Description: st.Description, Name: st.Name, StatusCategory: st.StatusCategory}
		return view, apiResp, nil
	}

	ReadResource[*workflowStatusStateView](ctx, req, resp, &data, readFn)

}

func (r *workflowStatusResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Update)
	defer cancel()
	var data workflowStatusResourceModel

	updateFn := func(ctx context.Context, id string, payload *models.WorkflowStatusPayloadScheme) (*workflowStatusStateView, *models.ResponseScheme, error) {
		apiResp, err := r.workflowStatusService.Update(ctx, payload)
		if err != nil {
			return nil, apiResp, err
		}
		statuses, apiResp2, err2 := r.workflowStatusService.Gets(ctx, []string{id}, nil)
		if err2 != nil {
			return nil, apiResp2, err2
		}
		if len(statuses) != 1 {
			return nil, apiResp2, fmt.Errorf("expected one status after update, got %d", len(statuses))
		}
		st := statuses[0]
		view := &workflowStatusStateView{ID: st.ID, Description: st.Description, Name: st.Name, StatusCategory: st.StatusCategory}
		return view, apiResp2, nil
	}

	UpdateResource[*models.WorkflowStatusPayloadScheme, *workflowStatusStateView](ctx, req, resp, &data, updateFn)

}

func (r *workflowStatusResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Delete)
	defer cancel()
	var data workflowStatusResourceModel

	deleteFn := func(ctx context.Context, id string) (*models.ResponseScheme, error) {
		return r.workflowStatusService.Delete(ctx, []string{id})
	}

	DeleteResource(ctx, req, resp, &data, deleteFn)
}

func (r *workflowStatusResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()
	var data workflowStatusResourceModel

	getFn := func(ctx context.Context, id string) (*workflowStatusStateView, *models.ResponseScheme, error) {
		statuses, apiResp, err := r.workflowStatusService.Gets(ctx, []string{id}, nil)
		if err != nil {
			return nil, apiResp, err
		}
		if len(statuses) != 1 {
			return nil, apiResp, fmt.Errorf("expected one status from import, got %d", len(statuses))
		}
		st := statuses[0]
		view := &workflowStatusStateView{ID: st.ID, Description: st.Description, Name: st.Name, StatusCategory: st.StatusCategory}
		return view, apiResp, nil
	}

	ImportResource[*workflowStatusStateView](ctx, req, resp, &data, getFn)
}
