package provider

import (
	"context"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type workTypeResourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	IconURL        types.String `tfsdk:"icon_url"`
	Subtask        types.Bool   `tfsdk:"subtask"`
	AvatarID       types.Int64  `tfsdk:"avatar_id"`
	HierarchyLevel types.Int32  `tfsdk:"hierarchy_level"`
}

func (i *workTypeResourceModel) GetApiPayload(_ context.Context) (createPayload *models.IssueTypePayloadScheme, diags diag.Diagnostics) {
	return &models.IssueTypePayloadScheme{
		Name:           i.Name.ValueString(),
		Description:    i.Description.ValueString(),
		HierarchyLevel: int(i.HierarchyLevel.ValueInt32()),
	}, nil
}

func (i *workTypeResourceModel) GetID() string {
	return i.Id.ValueString()
}

func (i *workTypeResourceModel) TransformToState(_ context.Context, issueType *models.IssueTypeScheme) diag.Diagnostics {
	resourceModel := workTypeResourceModel{
		Id:             types.StringValue(issueType.ID),
		Name:           types.StringValue(issueType.Name),
		IconURL:        types.StringValue(issueType.IconURL),
		Subtask:        types.BoolValue(issueType.Subtask),
		AvatarID:       types.Int64Value(int64(issueType.AvatarID)),
		HierarchyLevel: types.Int32Value(int32(issueType.HierarchyLevel)),
	}

	if issueType.Description != "" {
		resourceModel.Description = types.StringValue(issueType.Description)
	}

	*i = resourceModel

	return nil
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
