// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type workTypeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	IconURL        types.String `tfsdk:"icon_url"`
	Subtask        types.Bool   `tfsdk:"subtask"`
	AvatarID       types.Int64  `tfsdk:"avatar_id"`
	HierarchyLevel types.Int32  `tfsdk:"hierarchy_level"`
}

func (i *workTypeResourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":              types.StringType,
		"name":            types.StringType,
		"description":     types.StringType,
		"icon_url":        types.StringType,
		"subtask":         types.BoolType,
		"avatar_id":       types.Int64Type,
		"hierarchy_level": types.Int32Type,
	}
}

// mapWorkTypeSchemeToModel centralizes mapping for resources/data sources and matches CRUDHooks MapToState signature.
func mapWorkTypeSchemeToModel(_ context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	*st = workTypeResourceModel{
		ID:             types.StringValue(api.ID),
		Name:           types.StringValue(api.Name),
		IconURL:        types.StringValue(api.IconURL),
		Subtask:        boolValue(api.Subtask),
		AvatarID:       types.Int64Value(int64(api.AvatarID)),
		HierarchyLevel: types.Int32Value(int32(api.HierarchyLevel)),
		Description:    stringOrNull(api.Description),
	}
	return diags
}
