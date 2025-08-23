resource "jira_project" "example" {
  key              = "ENG"
  name             = "Engineering"
  project_type_key = "software"
  lead_account_id  = "abc123" # REQUIRED: replace with a valid Jira account ID for the project lead

  # Optional fields
  # description     = "Engineering project"
  # assignee_type   = "PROJECT_LEAD"
  # category_id     = 10000
}
