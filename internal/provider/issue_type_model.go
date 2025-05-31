package provider

import (
	"context"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type issueTypeResourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	IconURL        types.String `tfsdk:"icon_url"`
	Subtask        types.Bool   `tfsdk:"subtask"`
	AvatarID       types.Int64  `tfsdk:"avatar_id"`
	HierarchyLevel types.Int32  `tfsdk:"hierarchy_level"`
}

func (i *issueTypeResourceModel) GetApiPayload(_ context.Context) (createPayload *models.IssueTypePayloadScheme, diags diag.Diagnostics) {
	return &models.IssueTypePayloadScheme{
		Name:           i.Name.ValueString(),
		Description:    i.Description.ValueString(),
		HierarchyLevel: int(i.HierarchyLevel.ValueInt32()),
	}, nil
}

func (i *issueTypeResourceModel) GetID() string {
	return i.Id.ValueString()
}

func (i *issueTypeResourceModel) TransformToState(_ context.Context, issueType *models.IssueTypeScheme) diag.Diagnostics {
	*i = issueTypeResourceModel{
		Id:             types.StringValue(issueType.ID),
		Name:           types.StringValue(issueType.Name),
		Description:    types.StringValue(issueType.Description),
		IconURL:        types.StringValue(issueType.IconURL),
		Subtask:        types.BoolValue(issueType.Subtask),
		AvatarID:       types.Int64Value(int64(issueType.AvatarID)),
		HierarchyLevel: types.Int32Value(int32(issueType.HierarchyLevel)),
	}

	return nil
}
