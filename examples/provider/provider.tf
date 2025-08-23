# Minimal provider configuration
# Credentials are sourced from environment variables:
#   JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
# For self-hosted Jira (basic auth): JIRA_USERNAME, JIRA_PASSWORD

terraform {
  required_providers {
    jira = {
      source = "devops-wiz/jira"
      # version constraints can be added by end users
    }
  }
}

provider "jira" {}
