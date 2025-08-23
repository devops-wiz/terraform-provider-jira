# Filter projects by string IDs

variable "project_ids" {
  type        = list(string)
  description = "Project string IDs to include"
  default     = ["10000", "10001"]
}

data "jira_projects" "by_ids" {
  ids = var.project_ids
}
