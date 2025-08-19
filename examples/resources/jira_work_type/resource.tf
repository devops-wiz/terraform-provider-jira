# Minimal example: create a Jira work type
# Note: For Standard Jira instances, omit hierarchy_level (or use -1 for subtask, 0 for standard).
# Credentials are expected via environment variables.

resource "jira_work_type" "example" {
  name        = "Example Work Type"
  description = "Created by Terraform example"
}
