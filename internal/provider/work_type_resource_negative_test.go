// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Negative scenarios for jira_work_type resource
func TestAccWorkTypeResource_negative(t *testing.T) {
	// Acceptance-gated via TF_ACC; these fail fast on validation and should not create resources in Standard Jira.
	t.Parallel()

	rName := "jira_work_type.test"

	t.Run("invalid hierarchy_level on Standard should fail validation", func(t *testing.T) {
		name := acctest.RandomWithPrefix("tf-acc-work-type-neg")
		// Direct config with invalid hierarchy level 2 (Standard only allows 0 or -1)
		cfg := fmt.Sprintf(`
resource "jira_work_type" "test" {
  name            = "%s"
  description     = "neg test"
  hierarchy_level = 2
}
`, name)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile(`(?s)Hierarchy Level.*not supported`),
				},
			},
		})
	})

	t.Run("import with bogus ID should return not found error", func(t *testing.T) {
		cfg := fmt.Sprintf(`
resource "jira_work_type" "test" {
  name            = "tf-acc-work-type-neg-import-%s"
  description     = "placeholder"
}
`, RandString(6))

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:        cfg,
					ResourceName:  rName,
					ImportState:   true,
					ImportStateId: "9999999999-bogus",
					ExpectError:   regexp.MustCompile(`(?s)Error: read imported resource failed`),
				},
			},
		})
	})
}
