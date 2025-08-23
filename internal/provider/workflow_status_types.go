// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type workflowStatusResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
	Name           types.String `tfsdk:"name"`
	StatusCategory types.String `tfsdk:"status_category"`
}

type workflowStatusStateView struct {
	ID             string
	Description    string
	Name           string
	StatusCategory string
}
