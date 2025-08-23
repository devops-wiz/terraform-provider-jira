data "jira_project" "example" {
  id = "10000"
}

output "project_key" {
  value = data.jira_project.example.project_key
}
