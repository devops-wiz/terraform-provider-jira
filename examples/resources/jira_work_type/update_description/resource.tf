# Update example: change the description of a Jira work type by editing and re-applying.
# Initial apply creates the resource; subsequent applies with a new description will update it.

resource "jira_work_type" "update_desc" {
  name        = "Example Work Type"
  description = "Initial description"
}
