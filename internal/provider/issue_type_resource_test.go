package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"testing"
)

func TestAccIssueTypeResource_basic(t *testing.T) {
	t.Parallel()
	rName := "jira_issue_type.test"

	t.Run("create standard issue type", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(resourceName, "Test issue type", 0),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
					},
				},
			},
		})
	})

	t.Run("create subtask issue type", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(resourceName, "Test issue type", -1),
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

	t.Run("should update an issue type description", func(t *testing.T) {
		resourceName := acctest.RandomWithPrefix("tfacc-issue-type")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

			Steps: []resource.TestStep{
				{
					Config: testAccIssueTypeResourceConfig(resourceName, "Test issue type", 0),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
					},
				},
				{
					Config: testAccIssueTypeResourceConfig(resourceName, "Test issue type 2", 0),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
					},
				},
			},
		})
	})

}

func testAccIssueTypeResourceConfig(name string, desc string, hierarchLvl int) string {
	return fmt.Sprintf(`
resource "jira_issue_type" "test" {
	name = "%s"
	description = "%s"
	hierarchy_level = %d
}
`, name, desc, hierarchLvl)
}
