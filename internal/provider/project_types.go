// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	ProjectTypeKey types.String `tfsdk:"project_type_key"`
	Description    types.String `tfsdk:"description"`
	URL            types.String `tfsdk:"url"`
	AssigneeType   types.String `tfsdk:"assignee_type"`
	LeadAccountID  types.String `tfsdk:"lead_account_id"`
	CategoryID     types.Int64  `tfsdk:"category_id"`
}

// mapProjectSchemeToModel centralizes mapping for resources and data sources and matches CRUDHooks MapToState signature.
func mapProjectSchemeToModel(_ context.Context, api *models.ProjectScheme, st *projectResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var catID int64
	if api.Category != nil && api.Category.ID != "" {
		if v, err := strconv.ParseInt(api.Category.ID, 10, 64); err == nil {
			catID = v
		}
	}
	*st = projectResourceModel{
		ID:             types.StringValue(api.ID),
		Key:            types.StringValue(api.Key),
		Name:           types.StringValue(api.Name),
		ProjectTypeKey: types.StringValue(api.ProjectTypeKey),
		Description:    stringOrNull(api.Description),
		URL:            urlOrNull(api.URL),
		AssigneeType:   stringOrNull(api.AssigneeType),
		LeadAccountID: func() types.String {
			if api.Lead != nil && api.Lead.AccountID != "" {
				return types.StringValue(api.Lead.AccountID)
			}
			return types.StringNull()
		}(),
		CategoryID: int64OrNull(catID),
	}
	return diags
}
