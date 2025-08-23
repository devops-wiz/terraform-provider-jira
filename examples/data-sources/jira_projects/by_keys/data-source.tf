# Filter projects by keys

variable "project_keys" {
  type        = list(string)
  description = "Project keys to include"
  default     = ["EXAMPLE", "ANOTHER"]
}

data "jira_projects" "by_keys" {
  keys = var.project_keys
}
