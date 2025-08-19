# Import by identity example for jira_workflow_status
# For jira_workflow_status, the identity is the canonical string ID.
# After running:
#   terraform import jira_workflow_status.example <STATUS_ID>
# run `terraform plan` to review imported state.

resource "jira_workflow_status" "example" {
  # Values here are placeholders and may be overridden by server state after import
  name            = "Imported Status"
  status_category = "TODO"
  description     = "Managed via Terraform after import"
}
