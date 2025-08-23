provider "jira" {
  retry_max_attempts       = 6
  retry_initial_backoff_ms = 750
  retry_max_backoff_ms     = 8000
}
