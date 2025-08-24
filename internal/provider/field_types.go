// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/constants"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fieldResourceModel represents the Terraform schema model for jira_field.
type fieldResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	FieldType      types.String `tfsdk:"field_type"`
	TrashOnDestroy types.Bool   `tfsdk:"trash_on_destroy"`
}

// GetAPIPayload converts the Terraform plan into the API payload for creating/updating a field.
func (m *fieldResourceModel) GetAPIPayload(_ context.Context) (createPayload *models.CustomFieldScheme, diags diag.Diagnostics) {
	if mapVal, ok := constants.FieldTypesMap[m.FieldType.ValueString()]; ok {
		createPayload = &models.CustomFieldScheme{
			Name:        m.Name.ValueString(),
			Description: m.Description.ValueString(),
			FieldType:   mapVal.Value,
			SearcherKey: mapVal.SearcherKey,
		}
		return createPayload, diags
	} else {
		diags = diag.Diagnostics{}
		diags.AddAttributeError(path.Root("field_type"), "Invalid Field Type", fmt.Sprintf("Field type: %s is not valid. Valid types include:\n%s", m.FieldType.ValueString(), strings.Join(constants.FieldTypeKeys, "\n")))
		return createPayload, diags
	}

}

// GetID returns the stable identifier of the field.
func (m *fieldResourceModel) GetID() string { return m.ID.ValueString() }

// TransformToState maps the API model into Terraform state. Since the Fields GET/Search API
// doesn’t echo back FieldType, we keep the previously loaded state value for that
// attribute (it is ForceNew and should not drift post-create).
func (m *fieldResourceModel) TransformToState(_ context.Context, apiModel *models.IssueFieldScheme) diag.Diagnostics {
	if apiModel == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Empty API model", "The Jira API returned no field payload to map into state.")}
	}

	newState := fieldResourceModel{
		ID:             types.StringValue(apiModel.ID),
		Name:           types.StringValue(apiModel.Name),
		FieldType:      types.StringValue(constants.GetFieldTypeShort(apiModel.Schema.Custom)),
		Description:    m.Description,
		TrashOnDestroy: m.TrashOnDestroy,
	}
	if apiModel.Description != "" {
		newState.Description = types.StringValue(apiModel.Description)
	}
	*m = newState
	return nil
}
