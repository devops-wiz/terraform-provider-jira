// SPDX-License-Identifier: MPL-2.0

package provider

// This file intentionally instantiates generic runners and helpers with all
// currently supported resource/data-source types to ensure the constrained
// generics remain compatible across refactors. It does not execute any logic;
// it only needs to compile.

import (
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// CRUDRunner instantiations (state, payload, api)
var (
	_ CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	_ CRUDRunner[projectResourceModel, *models.ProjectPayloadScheme, *models.ProjectScheme]
	_ CRUDRunner[projectCategoryResourceModel, *models.ProjectCategoryPayloadScheme, *models.ProjectCategoryScheme]
)

// ListHooks instantiations (api list item, out model)
var (
	_ ListHooks[*models.IssueTypeScheme, workTypeResourceModel]
	_ ListHooks[*models.ProjectScheme, projectResourceModel]
	_ ListHooks[*models.ProjectCategoryScheme, projectCategoryResourceModel]
)
