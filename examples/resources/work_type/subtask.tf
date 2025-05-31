resource "jira_work_type" "subtask_example" {
  name = "Example Subtask Issue Type"
  # Optional
  description = "This is an example work type"
  # 0 by default for subtask
  hierarchy_level = -1
}
