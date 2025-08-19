resource "jira_work_type" "subtask_example" {
  name = "Example Subtask Issue Type"
  # Optional
  description = "This is an example work type"
  # -1 for subtask
  hierarchy_level = -1
}
