# Filter work types by IDs
# The data source returns a map keyed by ID; providing IDs narrows the result set.
# Replace the example IDs with real IDs from your Jira instance.

data "jira_work_types" "by_ids" {
  ids = [
    "10000",
    "10004",
  ]
}
