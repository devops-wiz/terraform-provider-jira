resource "jira_work_type" "standard_example" {
  name = "Example Issue Type"
  # Optional
  description = "This is an example work type"
  # 0 by default for standard
  hierarchy_level = 0
}
