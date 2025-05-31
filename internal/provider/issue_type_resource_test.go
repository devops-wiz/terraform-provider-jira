package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"testing"
)

var (
	standardIssueType = 0
	subtaskIssueType  = -1
)

func TestAccIssueTypeResource_basic(t *testing.T) {
	t.Parallel()
	rName := "jira_issue_type.test"

	t.Run("create standard issue type with hierarchy_level default", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type", nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
			},
		})
	})
	t.Run("create standard issue type with hierarchy_level set", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type", &standardIssueType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
			},
		})
	})

	t.Run("create subtask issue type", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type", &subtaskIssueType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(-1)),
					},
				},
			},
		})
	})

}

func TestAccIssueTypeResource_update(t *testing.T) {
	t.Parallel()
	rName := "jira_issue_type.test"

	t.Run("update an issue type description and import", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type", nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type 2", nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					// Test Step 3: import mode
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ResourceName:    rName,
				},
			},
		})
	})
	t.Run("recreate when changing Hierarchy level", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type", &standardIssueType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					Config: testAccIssueTypeResourceConfig(t, resourceName, "Test issue type 2", &subtaskIssueType),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(rName, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(-1)),
					},
				},
			},
		})
	})

}

func testAccIssueTypeResourceConfig(t *testing.T, name string, desc string, hierarchLvl *int) string {
	t.Helper()
	if hierarchLvl == nil {
		return fmt.Sprintf(`
resource "jira_issue_type" "test" {
	name = "%s"
	description = "%s"
}
`, name, desc)
	} else {
		return fmt.Sprintf(`
resource "jira_issue_type" "test" {
	name = "%s"
	description = "%s"
	hierarchy_level = %d
}
`, name, desc, *hierarchLvl)
	}
}
