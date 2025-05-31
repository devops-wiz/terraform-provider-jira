resource "jira_issue_type" "standard_example" {
  name = "Example Issue Type"
  # Optional
  description = "This is an example issue type"
  # 0 by default for standard
  hierarchy_level = 0
}

resource "jira_issue_type" "subtask_example" {
  name = "Example Subtask Issue Type"
  # Optional
  description = "This is an example issue type"
  # 0 by default for subtask
  hierarchy_level = -1
}
