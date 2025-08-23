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

func (m *projectResourceModel) GetAPIPayload(_ context.Context) (*models.ProjectPayloadScheme, diag.Diagnostics) {
	payload := &models.ProjectPayloadScheme{
		Key:            m.Key.ValueString(),
		Name:           m.Name.ValueString(),
		ProjectTypeKey: m.ProjectTypeKey.ValueString(),
		Description:    m.Description.ValueString(),
		AssigneeType:   m.AssigneeType.ValueString(),
		LeadAccountID:  m.LeadAccountID.ValueString(),
	}
	if !m.CategoryID.IsNull() && !m.CategoryID.IsUnknown() {
		payload.CategoryID = int(m.CategoryID.ValueInt64())
	}
	return payload, nil
}

func (m *projectResourceModel) GetID() string { return m.ID.ValueString() }

// TransformToState maps the Jira ProjectScheme directly into Terraform state without an intermediate view.
func (m *projectResourceModel) TransformToState(_ context.Context, p *models.ProjectScheme) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if p == nil {
		diags.AddError("Unexpected empty API model when mapping project", "The Jira API returned no project payload to map into state.")
		return diags
	}
	m.ID = types.StringValue(p.ID)
	m.Key = types.StringValue(p.Key)
	m.Name = types.StringValue(p.Name)
	m.ProjectTypeKey = types.StringValue(p.ProjectTypeKey)

	if p.Description != "" {
		m.Description = types.StringValue(p.Description)
	} else {
		m.Description = types.StringNull()
	}
	if p.URL != "" {
		m.URL = types.StringValue(p.URL)
	} else {
		m.URL = types.StringNull()
	}
	if p.AssigneeType != "" {
		m.AssigneeType = types.StringValue(p.AssigneeType)
	} else {
		m.AssigneeType = types.StringNull()
	}
	if p.Lead != nil && p.Lead.AccountID != "" {
		m.LeadAccountID = types.StringValue(p.Lead.AccountID)
	} else {
		m.LeadAccountID = types.StringNull()
	}
	var catID int64
	if p.Category != nil && p.Category.ID != "" {
		if v, err := strconv.ParseInt(p.Category.ID, 10, 64); err == nil {
			catID = v
		}
	}
	if catID != 0 {
		m.CategoryID = types.Int64Value(catID)
	} else {
		m.CategoryID = types.Int64Null()
	}
	return diags
}
