// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectCategoriesDataSource_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "jira_project_categories" "test" { }`,
			},
		},
	})
}

func TestAccProjectCategoriesDataSource_filterByNameAndID(t *testing.T) {
	t.Parallel()
	rName := "data.jira_project_categories.by_name"
	rNameID := "data.jira_project_categories.by_id"

	catName := acctest.RandomWithPrefix("tf-acc-category-ds")
	catName = strings.ReplaceAll(catName, "_", "-")
	cfg := fmt.Sprintf(`
 resource "jira_project_category" "c" {
  name        = "%s"
  description = "managed by acceptance test"
}

data "jira_project_categories" "by_name" {
  names = [jira_project_category.c.name]
}

data "jira_project_categories" "by_id" {
  ids = [jira_project_category.c.id]
}
`, catName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("categories"), knownvalue.MapSizeExact(1)),
					statecheck.ExpectKnownValue(rNameID, tfjsonpath.New("categories"), knownvalue.MapSizeExact(1)),
				},
			},
		},
	})
}

func TestAccProjectCategoriesDataSource_unsupportedArgument(t *testing.T) {
	t.Parallel()
	// Using an unsupported argument should fail schema validation before making any API calls.
	cfg := `
data "jira_project_categories" "invalid" {
  foo = "bar"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				// Match common Terraform errors for unknown/unsupported arguments.
				ExpectError: regexp.MustCompile(`(?s)(Unsupported argument|not expected here)`),
			},
		},
	})
}
