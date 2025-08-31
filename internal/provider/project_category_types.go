// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// projectCategoryResourceModel models the Terraform schema/state for jira_project_category.
type projectCategoryResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (m *projectCategoryResourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
	}
}

// mapProjectCategorySchemeToModel centralizes mapping for resources/data sources and matches CRUDHooks MapToState signature.
func mapProjectCategorySchemeToModel(_ context.Context, api *models.ProjectCategoryScheme, st *projectCategoryResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	*st = projectCategoryResourceModel{
		ID:          types.StringValue(api.ID),
		Name:        types.StringValue(api.Name),
		Description: stringOrNull(api.Description),
	}
	return diags
}
