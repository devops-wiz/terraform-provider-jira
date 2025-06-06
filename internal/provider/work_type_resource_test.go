package provider

import (
	"bytes"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"testing"
	"text/template"
)

const defaultWorkTypeDescription = "Default Work Type Description"

var baseWorkType = models.IssueTypePayloadScheme{
	Description: defaultWorkTypeDescription,
}

func TestAccWorkTypeResource_basic(t *testing.T) {
	t.Parallel()
	rName := "jira_work_type.test"

	t.Run("create standard work type with empty description", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		workType.Description = ""
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
			},
		})
	})

	t.Run("create standard work type with hierarchy_level default", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
			},
		})
	})

	t.Run("create standard work type with hierarchy_level set", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		workType.HierarchyLevel = standardWorkType
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
			},
		})
	})

	t.Run("create subtask work type", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		workType.HierarchyLevel = subtaskWorkType
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(-1)),
					},
				},
			},
		})
	})

}

func TestAccWorkTypeResource_update(t *testing.T) {
	t.Parallel()
	rName := "jira_work_type.test"

	t.Run("update an work type description and import", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		workType.HierarchyLevel = standardWorkType
		workTypeChanged := workType
		updatedDescription := "Updated Work Type Description"
		workTypeChanged.Description = updatedDescription
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(defaultWorkTypeDescription)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					Config: testAccWorkTypeResourceConfig(t, workTypeChanged),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("description"), knownvalue.StringExact(updatedDescription)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ResourceName:    rName,
				},
			},
		})
	})
	t.Run("recreate when changing Hierarchy level", func(t *testing.T) {
		t.Parallel()
		resourceName := acctest.RandomWithPrefix("tf-acc-work-type")
		workType := baseWorkType
		workType.Name = resourceName
		workType.HierarchyLevel = standardWorkType
		workTypeChanged := workType
		workTypeChanged.HierarchyLevel = subtaskWorkType
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccWorkTypeResourceConfig(t, workType),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(0)),
					},
				},
				{
					Config: testAccWorkTypeResourceConfig(t, workTypeChanged),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(rName, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("name"), knownvalue.StringExact(resourceName)),
						statecheck.ExpectKnownValue(rName, tfjsonpath.New("hierarchy_level"), knownvalue.Int32Exact(-1)),
					},
				},
			},
		})
	})

}

func testAccWorkTypeResourceConfig(t *testing.T, workType models.IssueTypePayloadScheme) string {
	t.Helper()
	tmpl, err := template.New(workTypeTmpl).ParseFiles(workTypeTmplPath)
	if err != nil {
		t.Fatal(err)
	}

	var tfFile bytes.Buffer

	err = tmpl.Execute(&tfFile, workType)

	if err != nil {
		t.Fatal(err)
	}

	return tfFile.String()
}
