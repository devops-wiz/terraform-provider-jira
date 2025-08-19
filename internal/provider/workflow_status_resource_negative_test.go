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

// Negative scenarios for jira_workflow_status resource
func TestAccWorkflowStatusResource_negative(t *testing.T) {
	// These tests require TF_ACC=1 to run, but they should fail fast during validation and not create resources.
	t.Parallel()

	rName := "jira_workflow_status.test"

	t.Run("invalid status_category should fail validation", func(t *testing.T) {
		name := acctest.RandomWithPrefix("tf-acc-wf-status-neg")
		badCategory := "START"
		cfg := fmt.Sprintf(`
resource "jira_workflow_status" "test" {
  name            = "%s"
  status_category = "%s"
  description     = "neg test"
}
`, name, badCategory)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      cfg,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*status_category`),
				},
			},
		})
	})

	t.Run("import with bogus ID should return not found error", func(t *testing.T) {
		name := acctest.RandomWithPrefix("tf-acc-wf-status-neg-import")
		cfg := fmt.Sprintf(`
resource "jira_workflow_status" "test" {
  name            = "%s"
  status_category = "TODO"
  description     = "placeholder"
}
`, name)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:        cfg,
					ResourceName:  rName,
					ImportState:   true,
					ImportStateId: "9999999999-bogus",
					// Summary should be from EnsureSuccessOrDiag: "Failed to read imported resource"
					ExpectError: regexp.MustCompile(`(?s)Error: read imported resource failed`),
				},
			},
		})
	})
}

// RandString provides a simple deterministic-ish suffix for names when acctest isn't required.
// We prefer acctest.RandomWithPrefix elsewhere; here we keep it self-contained.
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[i%len(letters)]
	}
	return string(s)
}
