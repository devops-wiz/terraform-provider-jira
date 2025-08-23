# Minimal example: list Jira work types
# Note: The work_types attribute is a map keyed by work type ID for stability across renames.

data "jira_work_types" "all" {}
