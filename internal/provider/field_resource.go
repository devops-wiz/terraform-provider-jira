// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/ctreminiom/go-atlassian/v2/service/jira"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/constants"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// _ is used to enforce that fieldResource implements the resource.Resource interface at compile time.
var _ resource.Resource = (*fieldResource)(nil)

// _ is used to enforce that `fieldResource` implements the `resource.ResourceWithConfigure` interface at compile time.
var _ resource.ResourceWithConfigure = (*fieldResource)(nil)

// _ is a compile-time assertion ensuring fieldResource implements resource.ResourceWithImportState interface.
var _ resource.ResourceWithImportState = (*fieldResource)(nil)

// NewFieldResource creates and returns a new instance of the fieldResource, representing a Jira custom field resource.
func NewFieldResource() resource.Resource { return &fieldResource{} }

// fieldResource represents a Terraform resource responsible for managing Jira custom fields globally.
type fieldResource struct {
	ServiceClient
	fieldService      jira.FieldConnector
	fieldTrashService jira.FieldTrashConnector
}

// Metadata sets the type name for the resource using the provider's type name concatenated with "_field".
func (r *fieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_field"
}

// Configure sets up the fieldResource by initializing its client, fieldService, and provider-specific timeouts.
func (r *fieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.fieldService = provider.client.Issue.Field
	r.fieldTrashService = provider.client.Issue.Field.Trash
	r.providerTimeouts = provider.providerTimeouts
}

// hooks wires fieldResource to the generic CRUD runner.
func (r *fieldResource) hooks() CRUDHooks[fieldResourceModel, *models.CustomFieldScheme, *models.IssueFieldScheme] {
	return CRUDHooks[fieldResourceModel, *models.CustomFieldScheme, *models.IssueFieldScheme]{
		BuildPayload: func(ctx context.Context, st *fieldResourceModel) (*models.CustomFieldScheme, diag.Diagnostics) {
			return st.GetAPIPayload(ctx)
		},
		APICreate: r.fieldService.Create,
		APIRead:   r.lookupFieldByID,
		APIUpdate: r.updateField,
		APIDelete: r.deleteField,
		ExtractID: func(st *fieldResourceModel) string { return st.GetID() },
		MapToState: func(ctx context.Context, api *models.IssueFieldScheme, st *fieldResourceModel) diag.Diagnostics {
			return st.TransformToState(ctx, api)
		},
		// Status handling uses defaults; no special overrides required for fields.
	}
}

// Schema defines the schema for the Jira custom field resource, including attributes and their validation requirements.
func (r *fieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Jira custom field (global).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The unique identifier of the field (e.g., customfield_10001).",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the field.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the field.",
			},
			"field_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("The field type key.\n\t- Valid values:\n\t\t- %s", strings.Join(constants.FieldTypeKeys, "\n\t\t* ")),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(constants.FieldTypeKeys...),
				},
			},
			"trash_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "If set to `false` (default: `true`), the field will be fully deleted from API side " +
					"when terraform destroys the resource, as opposed to moving to the trash.",
				Default: booldefault.StaticBool(true),
			},
		},
	}
}

// Create handles the operation to create a new resource, applying a timeout and invoking the service's create method.
func (r *fieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Create)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoCreate(
		ctx,
		func(ctx context.Context, dst *fieldResourceModel) diag.Diagnostics { return req.Plan.Get(ctx, dst) },
		func(ctx context.Context, src *fieldResourceModel) diag.Diagnostics { return resp.State.Set(ctx, src) },
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

// lookupFieldByID searches for a Jira issue field by its ID with retries, due to eventual consistency in the Jira API.
// It scans the list of fields returned by the `Gets` method and returns the matching field, response, or an error.
// If the field is not found after a set number of retries, an error is returned.
func (r *fieldResource) lookupFieldByID(ctx context.Context, id string) (*models.IssueFieldScheme, *models.ResponseScheme, error) {
	// Jira API does not provide a direct get-by-ID in this client version; scan list with a short retry
	// window to accommodate eventual consistency right after create/update.
	var lastResp *models.ResponseScheme
	for attempts := 0; attempts < 6; attempts++ { // ~3s total with 500ms sleeps
		allFields, apiResp, err := r.fieldService.Gets(ctx)
		lastResp = apiResp
		if err != nil {
			return nil, apiResp, err
		}
		for _, field := range allFields {
			if field != nil && field.ID == id {
				return field, apiResp, nil
			}
		}
		// Not found; if context is done, stop, otherwise back off briefly
		select {
		case <-ctx.Done():
			return nil, apiResp, ctx.Err()
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil, lastResp, fmt.Errorf("field %s not found", id)
}

// Read retrieves the current state of the field resource and updates the Terraform state accordingly.
func (r *fieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoRead(
		ctx,
		func(ctx context.Context, dst *fieldResourceModel) diag.Diagnostics { return req.State.Get(ctx, dst) },
		func(ctx context.Context, src *fieldResourceModel) diag.Diagnostics { return resp.State.Set(ctx, src) },
		func(ctx context.Context) { resp.State.RemoveResource(ctx) },
		ensureWith(&resp.Diagnostics),
		HTTPStatusFromScheme,
	)
	resp.Diagnostics.Append(diags...)
}

// updateField updates mutable properties of a custom field by its ID in Jira. Non-mutable fields are ignored on update.
// Parameters: ctx (context.Context) for request scope and cancellation, id (string) to specify the field to update,
// and updatedResource (*models.CustomFieldScheme) containing the updated field data.
// Returns: the updated field (*models.IssueFieldScheme), the API response (*models.ResponseScheme), and an error if any.
func (r *fieldResource) updateField(ctx context.Context, id string, updatedResource *models.CustomFieldScheme) (*models.IssueFieldScheme, *models.ResponseScheme, error) {
	// Jira does not allow changing a field type on update; only mutable fields should be sent.
	u := &models.CustomFieldScheme{
		Name:        updatedResource.Name,
		Description: updatedResource.Description,
	}
	apiResp, err := r.fieldService.Update(ctx, id, u)
	if err != nil {
		return nil, apiResp, err
	}
	issueField, apiResp2, err := r.lookupFieldByID(ctx, id)
	if err != nil {
		return nil, apiResp2, err
	}
	issueField.Description = updatedResource.Description
	issueField.Name = updatedResource.Name
	return issueField, apiResp2, nil
}

// Update updates an existing resource by applying changes specified in the request and writes the updated state.
func (r *fieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Update)
	defer cancel()

	runner := NewCRUDRunner(r.hooks())
	diags := runner.DoUpdate(
		ctx,
		func(ctx context.Context, dst *fieldResourceModel) diag.Diagnostics { return req.Plan.Get(ctx, dst) },
		func(ctx context.Context, src *fieldResourceModel) diag.Diagnostics { return resp.State.Set(ctx, src) },
		ensureWith(&resp.Diagnostics),
	)
	resp.Diagnostics.Append(diags...)
}

// deleteField deletes a custom field in Jira using its unique identifier and returns the API response or an error.
func (r *fieldResource) deleteField(ctx context.Context, id string) (*models.ResponseScheme, error) {
	_, rs, err := r.fieldService.Delete(ctx, id)
	return rs, err
}

// Delete removes the specified field resource. If the resource does not exist, the operation is treated as successful.
func (r *fieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Delete)
	defer cancel()

	var trashOnDelete bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("trash_on_destroy"), &trashOnDelete)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if trashOnDelete {

		var id string

		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)

		if resp.Diagnostics.HasError() {
			return
		}

		apiResp, err := r.fieldTrashService.Move(ctx, id)

		if !EnsureSuccessOrDiagFromScheme(ctx, "trash", apiResp, err, &resp.Diagnostics) {
			return
		}

	} else {
		runner := NewCRUDRunner(r.hooks())

		resp.Diagnostics.Append(
			runner.DoDelete(
				ctx,
				func(ctx context.Context, dst *fieldResourceModel) diag.Diagnostics {
					return req.State.Get(ctx, dst)
				},
				ensureWith(&resp.Diagnostics),
			)...,
		)
	}

}

// ImportState imports the resource's state into Terraform by fetching the resource from the API using its ID.
func (r *fieldResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	ctx, cancel := withTimeout(ctx, r.providerTimeouts.Read)
	defer cancel()

	diags := DoImport[fieldResourceModel, *models.IssueFieldScheme](
		ctx,
		request.ID,
		r.lookupFieldByID,
		func(ctx context.Context, api *models.IssueFieldScheme, st *fieldResourceModel) diag.Diagnostics {
			return st.TransformToState(ctx, api)
		},
		func(ctx context.Context, src *fieldResourceModel) diag.Diagnostics {
			return response.State.Set(ctx, src)
		},
		ensureWith(&response.Diagnostics),
	)
	response.Diagnostics.Append(diags...)
}
