// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectCategoryResource_basic(t *testing.T) {
	t.Parallel()

	rName := "jira_project_category.test"
	name := acctest.RandomWithPrefix("tf-acc-category")
	updatedDesc := "Updated category description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.GetProjCatCfg(t, name, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
			{
				Config: testhelpers.GetProjCatCfg(t, name, updatedDesc),
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
