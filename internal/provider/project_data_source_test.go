// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceProject_lookupByKeyAndID(t *testing.T) {
	t.Parallel()

	// Create a project resource, then look it up via data source by key and by id
	key := randomProjectKey(6)
	name := acctest.RandomWithPrefix("tf-acc-ds-project")
	name = strings.ReplaceAll(name, "_", "-")
	projectType := "software"

	rName := "jira_project.test"
	dsByKey := "data.jira_project.by_key"
	dsByID := "data.jira_project.by_id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectWithDataSourceByKey(key, name, projectType),
				ConfigStateChecks: []statecheck.StateCheck{
					// resource exists
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("key"), knownvalue.StringExact(key)),
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(name)),
					// data source by key should resolve the same project
					statecheck.ExpectKnownValue(dsByKey, tfjsonpath.New("project_key"), knownvalue.StringExact(key)),
					statecheck.ExpectKnownValue(dsByKey, tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
			{
				Config: testAccProjectWithDataSourceByID(key, name, projectType),
				ConfigStateChecks: []statecheck.StateCheck{
					// data source by id should resolve and match the resource
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("project_key"), knownvalue.StringExact(key)),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
		},
	})
}

func testAccProjectWithDataSourceByKey(key, name, projectType string) string {
	return fmt.Sprintf(`
resource "jira_project" "test" {
  key               = "%s"
  name              = "%s"
  project_type_key  = "%s"
  lead_account_id   = "%s"
}

data "jira_project" "by_key" {
  key = jira_project.test.key
}
`, key, name, projectType, testAccLeadAccountID())
}

func testAccProjectWithDataSourceByID(key, name, projectType string) string {
	return fmt.Sprintf(`
resource "jira_project" "test" {
  key               = "%s"
  name              = "%s"
  project_type_key  = "%s"
  lead_account_id   = "%s"
}

data "jira_project" "by_id" {
  id = jira_project.test.id
}
`, key, name, projectType, testAccLeadAccountID())
}

func testAccLeadAccountID() string {
	return strings.TrimSpace(os.Getenv("JIRA_PROJECT_TEST_ROLE_LEAD_ID"))
}
