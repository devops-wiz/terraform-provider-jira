package provider

import (
	"context"
	"errors"
	"fmt"
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var _ resource.Resource = (*workflowStatusResource)(nil)
var _ resource.ResourceWithConfigure = (*workflowStatusResource)(nil)
var _ resource.ResourceWithImportState = (*workflowStatusResource)(nil)

func NewWorkflowStatusResource() resource.Resource {
	return &workflowStatusResource{}
}

type workflowStatusResource struct {
	client  *jira.Client
	premium bool
}

type workflowStatusResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
	Name           types.String `tfsdk:"name"`
	StatusCategory types.String `tfsdk:"status_category"`
}

func (r *workflowStatusResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.premium = provider.premium
}

func (r *workflowStatusResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_status"
}

func (r *workflowStatusResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the status.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the status.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the status.",
			},
			"status_category": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The category of the status. Valid values: `TODO`, `IN_PROGRESS`, `DONE`",
				Validators: []validator.String{
					stringvalidator.OneOf("TODO", "IN_PROGRESS", "DONE"),
				},
			},
		},
	}
}

func (r *workflowStatusResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data workflowStatusResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newStatus := &models.WorkflowStatusPayloadScheme{
		Scope: &models.WorkflowStatusScopeScheme{
			Type: "GLOBAL",
		},
		Statuses: []*models.WorkflowStatusNodeScheme{
			{
				Name:           data.Name.ValueString(),
				StatusCategory: data.StatusCategory.ValueString(),
				Description:    data.Description.ValueString(),
			},
		},
	}

	statusResp, apiResp, err := r.client.Workflow.Status.Create(ctx, newStatus)

	if err != nil {
		if errors.Is(err, models.ErrInvalidStatusCode) {
			if apiResp.Code != 409 {
				resp.Diagnostics.AddError("Error creating workflow status resource: Invalid response status", fmt.Sprintf("Code: %d", apiResp.Code))
				return
			} else {
				time.Sleep(3 * time.Second)
			}
		} else {
			resp.Diagnostics.AddError("Error creating workflow status resource", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
	}

	if apiResp.StatusCode != 200 {
		resp.Diagnostics.AddError("Error creating workflow status resource", fmt.Sprintf("Error: %s", apiResp.Bytes.String()))
		return
	}

	if len(statusResp) != 1 {
		resp.Diagnostics.AddError("Error creating workflow status resource", fmt.Sprintf("Error: Only one status should have been returned. %d was returned.", len(statusResp)))
		return
	}

	data = workflowStatusResourceModel{
		ID:             types.StringValue(statusResp[0].ID),
		Description:    types.StringValue(statusResp[0].Description),
		Name:           types.StringValue(statusResp[0].Name),
		StatusCategory: types.StringValue(statusResp[0].StatusCategory),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *workflowStatusResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var data workflowStatusResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	statusResp, apiResp, err := r.client.Workflow.Status.Gets(ctx, []string{data.ID.ValueString()}, nil)
	if err != nil {
		if errors.Is(err, models.ErrInvalidStatusCode) {
			resp.Diagnostics.AddError("Error reading workflow status resource: Invalid response status", fmt.Sprintf("Code: %d", apiResp.Code))
			return
		}
		resp.Diagnostics.AddError("Error reading workflow status resource", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if apiResp.StatusCode != 200 {
		resp.Diagnostics.AddError("Error reading workflow status resource", fmt.Sprintf("Error: %s", apiResp.Bytes.String()))
		return
	}

	if len(statusResp) != 1 {
		resp.Diagnostics.AddError("Error reading workflow status resource", fmt.Sprintf("Error: Only one status should have been returned. %d was returned.", len(statusResp)))
		return
	}

	data = workflowStatusResourceModel{
		ID:             types.StringValue(statusResp[0].ID),
		Description:    types.StringValue(statusResp[0].Description),
		Name:           types.StringValue(statusResp[0].Name),
		StatusCategory: types.StringValue(statusResp[0].StatusCategory),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *workflowStatusResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var data workflowStatusResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedStatus := &models.WorkflowStatusPayloadScheme{
		Statuses: []*models.WorkflowStatusNodeScheme{
			{
				ID:             data.ID.ValueString(),
				Name:           data.Name.ValueString(),
				StatusCategory: data.StatusCategory.ValueString(),
				Description:    data.Description.ValueString(),
			},
		},
	}

	apiResp, err := r.client.Workflow.Status.Update(ctx, updatedStatus)

	if err != nil {
		if errors.Is(err, models.ErrInvalidStatusCode) {
			if apiResp.Code != 409 {
				resp.Diagnostics.AddError("Error updating workflow status resource: Invalid response status", fmt.Sprintf("Code: %d", apiResp.Code))
				return
			} else {
				time.Sleep(3 * time.Second)
			}
		} else {
			resp.Diagnostics.AddError("Error updating workflow status resource", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
	}
	if apiResp.StatusCode != 204 {
		resp.Diagnostics.AddError("Error updating workflow status resource", fmt.Sprintf("Error: %s", apiResp.Bytes.String()))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

}

func (r *workflowStatusResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var modelID types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &modelID)...)

	apiResp, err := r.client.Workflow.Status.Delete(ctx, []string{modelID.ValueString()})

	if err != nil {
		if errors.Is(err, models.ErrInvalidStatusCode) {
			if apiResp.Code != 409 {
				resp.Diagnostics.AddError("Error deleting workflow status resource: Invalid response status", fmt.Sprintf("Code: %d", apiResp.Code))
				return
			} else {
				time.Sleep(3 * time.Second)
			}
		} else {
			resp.Diagnostics.AddError("Error deleting workflow status resource", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
	}

	if apiResp.StatusCode != 204 {
		resp.Diagnostics.AddError("Error deleting workflow status resource", fmt.Sprintf("Error: %s", apiResp.Bytes.String()))
		return
	}

}

func (r *workflowStatusResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	statusResp, apiResp, err := r.client.Workflow.Status.Gets(ctx, []string{req.ID}, nil)

	if err != nil {
		if errors.Is(err, models.ErrInvalidStatusCode) {
			if apiResp.Code != 409 {
				resp.Diagnostics.AddError("Error importing workflow status resource: Invalid response status", fmt.Sprintf("Code: %d", apiResp.Code))
				return
			} else {
				time.Sleep(3 * time.Second)
			}
		} else {
			resp.Diagnostics.AddError("Error importing workflow status resource", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
	}

	if apiResp.StatusCode != 200 {
		resp.Diagnostics.AddError("Error importing workflow status resource", fmt.Sprintf("Error: %s", apiResp.Bytes.String()))
		return
	}

	data := workflowStatusResourceModel{
		ID:             types.StringValue(statusResp[0].ID),
		Description:    types.StringValue(statusResp[0].Description),
		Name:           types.StringValue(statusResp[0].Name),
		StatusCategory: types.StringValue(statusResp[0].StatusCategory),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
