// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const defaultWorkflowStatusDescription = "Default Workflow Status Description"

var baseWorkflowStatus = models.WorkflowStatusNodeScheme{
	Description:    defaultWorkflowStatusDescription,
	StatusCategory: "TODO",
}

func TestAccWorkflowStatusResource_basic(t *testing.T) {
	t.Parallel()
	rName := "jira_workflow_status.test"

	t.Run("create workflow status with empty description", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tf-acc-workflow-status")
		workflowStatus := baseWorkflowStatus
		workflowStatus.Name = resourceName
		workflowStatus.Description = ""
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatus),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("TODO")),
					},
				},
			},
		})
	})

	t.Run("create workflow status with TODO category", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tf-acc-workflow-status")
		workflowStatus := baseWorkflowStatus
		workflowStatus.Name = resourceName
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatus),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("TODO")),
					},
				},
			},
		})
	})

	t.Run("create workflow status with IN_PROGRESS category", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tf-acc-workflow-status")
		workflowStatus := baseWorkflowStatus
		workflowStatus.Name = resourceName
		workflowStatus.StatusCategory = "IN_PROGRESS"
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatus),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("IN_PROGRESS")),
					},
				},
			},
		})
	})

	t.Run("create workflow status with DONE category", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tf-acc-workflow-status")
		workflowStatus := baseWorkflowStatus
		workflowStatus.Name = resourceName
		workflowStatus.StatusCategory = "DONE"
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatus),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("DONE")),
					},
				},
			},
		})
	})
}

func TestAccWorkflowStatusResource_update(t *testing.T) {
	t.Parallel()
	rName := "jira_workflow_status.test"

	t.Run("update workflow status description and import", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tf-acc-workflow-status")
		workflowStatus := baseWorkflowStatus
		workflowStatus.Name = resourceName
		workflowStatusChanged := workflowStatus
		updatedDescription := "Updated Workflow Status Description"
		workflowStatusChanged.Description = updatedDescription
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatus),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(defaultWorkflowStatusDescription)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("TODO")),
					},
				},
				{
					Config: testAccWorkflowStatusResourceConfig(workflowStatusChanged),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(updatedDescription)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("status_category"), knownvalue.StringExact("TODO")),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ResourceName:    rName,
				},
			},
		})
	})
}

func testAccWorkflowStatusResourceConfig(workflowStatus models.WorkflowStatusNodeScheme) string {
	return fmt.Sprintf(`
resource "jira_workflow_status" "test" {
  name            = "%s"
  status_category = "%s"
  description     = "%s"
}
`, workflowStatus.Name, workflowStatus.StatusCategory, workflowStatus.Description)
}
