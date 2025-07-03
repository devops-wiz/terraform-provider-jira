resource "jira_workflow_status" "test" {
  name            = "Example Status"
  status_category = "TODO"
  description     = "Test Description"
}
