# Filter projects using query and type_keys

variable "query" {
  type        = string
  description = "Filter string (matches key or name, case-insensitive)"
  default     = "test"
}

variable "type_keys" {
  type        = list(string)
  description = "Project type keys to include (e.g., software, service_desk, business)"
  default     = ["software"]
}

data "jira_projects" "query_and_type" {
  query     = var.query
  type_keys = var.type_keys
  # Optionally, order the results by a server-supported field
  # order_by = "key"
}
