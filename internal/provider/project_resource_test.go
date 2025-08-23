// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// randomProjectKey generates a Jira project key (uppercase letters) of length n (2..10)
// nolint:unparam // kept signature for potential future variation; currently called with constant length in tests
func randomProjectKey(n int) string {
	if n < 2 {
		n = 2
	}
	if n > 10 {
		n = 10
	}
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func TestAccProjectResource_basic(t *testing.T) {
	t.Parallel()

	rName := "jira_project.test"

	// Random but valid key and names
	key := randomProjectKey(6)
	name := acctest.RandomWithPrefix("tf-acc-project")
	// Ensure the name doesn't contain characters Jira forbids for project names
	name = strings.ReplaceAll(name, "_", "-")

	projectType := "software"
	updatedDesc := "Updated project description"
	leadAccountID := testhelpers.TestAccLeadAccountID()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.TestAccProjectResourceConfig(t, key, name, projectType, leadAccountID, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("key"), knownvalue.StringExact(key)),
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("project_type_key"), knownvalue.StringExact(projectType)),
				},
			},
			{
				Config: testhelpers.TestAccProjectResourceConfig(t, key, name, projectType, leadAccountID, updatedDesc),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(updatedDesc)),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ResourceName:    rName,
			},
		},
	})
}
