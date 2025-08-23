// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// projectCategoryResourceModel models the Terraform schema/state for jira_project_category
// and implements the ResourceTransformer contract for generic CRUD helpers.
type projectCategoryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (m *projectCategoryResourceModel) GetAPIPayload(_ context.Context) (createPayload *models.ProjectCategoryPayloadScheme, diags diag.Diagnostics) {
	return &models.ProjectCategoryPayloadScheme{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
	}, nil
}

func (m *projectCategoryResourceModel) GetID() string { return m.ID.ValueString() }

func (m *projectCategoryResourceModel) TransformToState(_ context.Context, cat *models.ProjectCategoryScheme) diag.Diagnostics {
	state := projectCategoryResourceModel{
		ID:   types.StringValue(cat.ID),
		Name: types.StringValue(cat.Name),
	}
	if cat.Description != "" {
		state.Description = types.StringValue(cat.Description)
	}
	*m = state
	return nil
}

func (m *projectCategoryResourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
	}
}
