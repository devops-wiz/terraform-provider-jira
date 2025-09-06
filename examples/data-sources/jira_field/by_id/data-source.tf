data "jira_field" "example" {
  id = "customfield_10001"
}

output "field_name" {
  value = data.jira_field.example.field_name
}

output "field_type" {
  value = data.jira_field.example.field_type
}