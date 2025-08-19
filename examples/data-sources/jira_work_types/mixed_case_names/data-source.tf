# Mixed-case name filtering demonstrates case-insensitive matching
# Replace the example names with those in your Jira instance (mixed cases shown).

data "jira_work_types" "mixed_case" {
  names = [
    "bug",
    "STORY",
  ]
}
