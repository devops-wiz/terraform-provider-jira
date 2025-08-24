

resource "jira_field" "example" {
  name        = "Team URL"
  field_type  = "com.atlassian.jira.plugin.system.customfieldtypes:url"
  description = "URL field for team homepage"
}
