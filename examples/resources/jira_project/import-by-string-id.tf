# Import by string/numeric ID example for jira_project
# After running:
#   terraform import jira_project.example <PROJECT_ID_OR_KEY>
# run `terraform plan` to review imported state.

resource "jira_project" "example" {
  # Values here are placeholders and may be overridden by server state after import
  key              = "ENG"
  name             = "Imported Project"
  project_type_key = "software"
  lead_account_id  = "abc123"
  description      = "Managed via Terraform after import"
}
