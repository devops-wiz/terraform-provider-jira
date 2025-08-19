# Advanced provider configuration: retries and timeouts
# Credentials are sourced from environment variables:
#   JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
# For self-hosted Jira (basic auth): JIRA_USERNAME, JIRA_PASSWORD

provider "jira" {
  # Per-attempt HTTP timeout (seconds)
  http_timeout_seconds = 60

  # Retry policy for 429/5xx
  retry_on_429_5xx         = true
  retry_max_attempts       = 6
  retry_initial_backoff_ms = 750
  retry_max_backoff_ms     = 8000

  # Optional CRUD operation timeouts
  operation_timeouts = {
    create = "2m"
    read   = "30s"
    update = "2m"
    delete = "2m"
  }
}
