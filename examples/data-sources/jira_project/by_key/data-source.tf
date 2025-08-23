data "jira_project" "example" {
  key = "EXAMPLE"
}

output "project_id" {
  value = data.jira_project.example.project_id
}
