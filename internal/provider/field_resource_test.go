// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
	"text/template"

	"maps"

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
		fieldType := "textfield" // use key as validated by schema
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

// TestAccFieldResource_variousTypes creates and imports fields for a selection of supported custom field types.
// To keep runtime reasonable, this test creates each type sequentially in its own step.
func TestAccFieldResource_variousTypes(t *testing.T) {
	t.Parallel()

	// Determine a deterministic, alphabetically-sorted set of field type keys.
	allKeys := slices.Sorted(maps.Keys(fieldTypesMap))

	// Optionally limit the number of tested types to avoid excessive runtime in CI. Adjust as needed.
	// Keeping full coverage by default since provider limits are modest; tweak slice below to reduce.
	keysToTest := allKeys

	for _, k := range keysToTest {
		// Capture loop variable
		typ := k

		t.Run(fmt.Sprintf("type=%s", typ), func(t *testing.T) {
			// Do not run subtests in parallel to avoid noisy flakiness/rate limits against Jira API.
			rName := "jira_field.test"
			name := acctest.RandomWithPrefix("tf-acc-field-" + typ)

			cfg := fieldTemplateData{
				Name:      name,
				FieldType: typ, // schema validates keys, provider maps to API value internally
			}

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccFieldResourceConfig(t, cfg),
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(name)),
							statecheck.ExpectKnownValue(rName, tfjsonpath.New("field_type"), knownvalue.StringExact(typ)),
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
