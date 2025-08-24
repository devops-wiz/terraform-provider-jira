// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
)

// fieldTemplateData represents the structure for storing field information such as name, type, and description.
type fieldTemplateData struct {
	Name        string
	FieldType   string
	Description string
}

// TestAccFieldResource_basic tests the creation, update, and import functionality of a Jira field resource.
func TestAccFieldResource_basic(t *testing.T) {
	t.Parallel()

	rName := "jira_field.test"

	t.Run("create field and update description, then import", func(t *testing.T) {
		t.Parallel()

		name := acctest.RandomWithPrefix("tf-acc-field")
		fieldType := "com.atlassian.jira.plugin.system.customfieldtypes:textfield"
		updatedDesc := "Updated field description"

		initial := fieldTemplateData{Name: name, FieldType: fieldType}
		changed := fieldTemplateData{Name: name, FieldType: fieldType, Description: updatedDesc}

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccFieldResourceConfig(t, initial),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(name)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("field_type"), knownvalue.StringExact(fieldType)),
					},
				},
				{
					Config: testAccFieldResourceConfig(t, changed),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(updatedDesc)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("field_type"), knownvalue.StringExact(fieldType)),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportCommandWithID,
					ResourceName:    rName,
				},
			},
		})
	})
}

// testAccFieldResourceConfig generates a Terraform field resource configuration using the provided template data.
func testAccFieldResourceConfig(t *testing.T, data fieldTemplateData) string {
	t.Helper()
	tmpl, err := template.New(testhelpers.FieldTmpl).ParseFiles(testhelpers.FieldTmplPath)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
