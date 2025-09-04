// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceProject_lookupByKeyAndID(t *testing.T) {
	t.Parallel()

	// Create a project resource, then look it up via data-source by key and by id
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
				Config: testhelpers.GetProjCfgWithDsTmpl(t, key, name, projectType, "key"),
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
				Config: testhelpers.GetProjCfgWithDsTmpl(t, key, name, projectType, "id"),
				ConfigStateChecks: []statecheck.StateCheck{
					// data source by id should resolve and match the resource
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("project_key"), knownvalue.StringExact(key)),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
		},
	})
}
