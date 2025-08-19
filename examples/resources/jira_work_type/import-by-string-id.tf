# Import by string ID example for jira_work_type
# After running:
#   terraform import jira_work_type.example <WORK_TYPE_ID>
# run `terraform plan` to review imported state.

resource "jira_work_type" "example" {
  # Values here are placeholders and may be overridden by server state after import
  name        = "Imported Work Type"
  description = "Managed via Terraform after import"
}
