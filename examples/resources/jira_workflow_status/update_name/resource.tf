# Update example: change the name of a Jira workflow status by editing and re-applying.
# Initial apply creates the resource; subsequent applies with a new name will update it.

resource "jira_workflow_status" "update_name" {
  name            = "Initial Status Name"
  status_category = "IN_PROGRESS"
  description     = "Initial description"
}
