data "jira_field" "example" {
  name = "Team URL"
}

output "field_id" {
  value = data.jira_field.example.field_id
}

output "field_type" {
  value = data.jira_field.example.field_type
}