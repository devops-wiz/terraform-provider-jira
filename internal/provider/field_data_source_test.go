// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceField_byID(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFieldByIDConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.jira_field.test", "field_name", "Summary"),
					resource.TestCheckResourceAttr("data.jira_field.test", "field_id", "summary"),
					resource.TestCheckResourceAttrSet("data.jira_field.test", "field_type"),
				),
			},
		},
	})
}

func TestAccDataSourceField_byName(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFieldByNameConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.jira_field.test", "field_name", "Summary"),
					resource.TestCheckResourceAttr("data.jira_field.test", "field_id", "summary"),
					resource.TestCheckResourceAttrSet("data.jira_field.test", "field_type"),
				),
			},
		},
	})
}

const testAccDataSourceFieldByIDConfig = `
data "jira_field" "test" {
  id = "summary"
}
`

const testAccDataSourceFieldByNameConfig = `
data "jira_field" "test" {
  name = "Summary"
}
`