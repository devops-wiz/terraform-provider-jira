// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectsDataSource_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jira_projects" "test" { }`,
			},
		},
	})
}

func TestAccProjectsDataSource_query(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jira_projects" "test" { query = "a" }`,
			},
		},
	})
}

// Create a project resource and filter by keys
func TestAccProjectsDataSource_filterByKeys(t *testing.T) {
	t.Parallel()
	rName := "data.jira_projects.test"

	key := randomProjectKey(6)
	name := acctest.RandomWithPrefix("tf-acc-projects-ds")
	name = strings.ReplaceAll(name, "_", "-")
	projectType := "software"

	cfg := fmt.Sprintf(`
resource "jira_project" "p" {
  key               = "%s"
  name              = "%s"
  project_type_key  = "%s"
  lead_account_id   = "%s"
}

data "jira_projects" "test" {
  keys = [jira_project.p.key]
}
`, key, name, projectType, testhelpers.GetTestProjLeadAcctIdFromEnv())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("projects"), knownvalue.MapSizeExact(1)),
				},
			},
		},
	})
}

// Create a project resource and filter by ids (string ID)
func TestAccProjectsDataSource_filterByIDs(t *testing.T) {
	t.Parallel()
	rName := "data.jira_projects.test"

	key := randomProjectKey(6)
	name := acctest.RandomWithPrefix("tf-acc-projects-ds")
	name = strings.ReplaceAll(name, "_", "-")
	projectType := "software"

	cfg := fmt.Sprintf(`
resource "jira_project" "p" {
  key               = "%s"
  name              = "%s"
  project_type_key  = "%s"
  lead_account_id   = "%s"
}

data "jira_projects" "test" {
  ids = [jira_project.p.id]
}
`, key, name, projectType, testhelpers.GetTestProjLeadAcctIdFromEnv())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("projects"), knownvalue.MapSizeExact(1)),
				},
			},
		},
	})
}
