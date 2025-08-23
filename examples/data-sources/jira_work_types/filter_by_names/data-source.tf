# Filter work types by names (case-insensitive matching)
# Replace the example names with those in your Jira instance.

data "jira_work_types" "by_names" {
  names = [
    "Bug",
    "Story",
  ]
}
