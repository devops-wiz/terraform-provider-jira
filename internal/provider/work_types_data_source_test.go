// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"regexp"
	"testing"
	"text/template"

	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var testTypeIDs = []string{"10000", "10004", "10002"}
var testTypeNames = []string{"Bug", "Story"}

func TestAccWorkTypesDataSource_basic(t *testing.T) {
	t.Parallel()
	rName := "data.jira_work_types.test"

	t.Run("read work types without arguments", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypesDataSourceConfig(t, nil, nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapPartial(map[string]knownvalue.Check{
							"10003": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10003"),
								"name": knownvalue.StringExact("Sub-task"),
							}),
						})),
					},
				},
			},
		})
	})

	t.Run("read work types with Ids argument", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypesDataSourceConfig(t, testTypeIDs, nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapPartial(map[string]knownvalue.Check{
							"10000": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10000"),
								"name": knownvalue.StringExact("Epic"),
							}),
							"10004": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10004"),
								"name": knownvalue.StringExact("Bug"),
							}),
							"10002": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10002"),
								"name": knownvalue.StringExact("Task"),
							}),
						})),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapSizeExact(3)),
					},
				},
			},
		})
	})

	t.Run("read work types with Names argument", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypesDataSourceConfig(t, nil, testTypeNames),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapPartial(map[string]knownvalue.Check{
							"10001": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10001"),
								"name": knownvalue.StringExact("Story"),
							}),
							"10004": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"id":   knownvalue.StringExact("10004"),
								"name": knownvalue.StringExact("Bug"),
							}),
						})),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapSizeExact(2)),
					},
				},
			},
		})
	})
}

func TestAccWorkTypesDataSource_expectError(t *testing.T) {
	t.Parallel()

	t.Run("conflicting attributes", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccWorkTypesDataSourceConfig(t, testTypeIDs, testTypeNames),
					ExpectError: regexp.MustCompile(`.*Error: Invalid Attribute Combination*`),
				},
			},
		})
	})
}

func testAccWorkTypesDataSourceConfig(t *testing.T, ids, names []string) string {
	t.Helper()

	tmpl, err := template.New(testhelpers.DataWorkTypesTmpl).ParseFiles(testhelpers.DataWorkTypesTmplPath)
	if err != nil {
		t.Fatal(err)
	}

	var tfFile bytes.Buffer

	config := struct {
		Ids   []string
		Names []string
	}{
		Ids:   ids,
		Names: names,
	}

	err = tmpl.Execute(&tfFile, config)

	if err != nil {
		t.Fatal(err)
	}

	return tfFile.String()
}

func TestAccWorkTypesDataSource_namesCaseInsensitive(t *testing.T) {
	t.Parallel()
	rName := "data.jira_work_types.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkTypesDataSourceConfig(t, nil, []string{"bUg", "STORY"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapPartial(map[string]knownvalue.Check{
						"10001": knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"id":   knownvalue.StringExact("10001"),
							"name": knownvalue.StringExact("Story"),
						}),
						"10004": knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"id":   knownvalue.StringExact("10004"),
							"name": knownvalue.StringExact("Bug"),
						}),
					})),
					statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapSizeExact(2)),
				},
			},
		},
	})
}

func TestAccWorkTypesDataSource_notFound(t *testing.T) {
	t.Parallel()
	rName := "data.jira_work_types.test"

	t.Run("names not found returns empty", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypesDataSourceConfig(t, nil, []string{"___DOES_NOT_EXIST___"}),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapSizeExact(0)),
					},
				},
			},
		})
	})

	t.Run("ids not found returns empty", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypesDataSourceConfig(t, []string{"99999999"}, nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("work_types"), knownvalue.MapSizeExact(0)),
					},
				},
			},
		})
	})
}
