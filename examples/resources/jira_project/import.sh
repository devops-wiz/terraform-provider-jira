#!/usr/bin/env bash
# Import a Jira project by its canonical ID or key.
# Usage:
#   terraform init    # ensure provider is initialized
#   terraform import jira_project.example <PROJECT_ID_OR_KEY>
# Examples:
#   terraform import jira_project.example 10001
#   terraform import jira_project.example ENG

set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "Usage: $0 <PROJECT_ID_OR_KEY>" >&2
  exit 1
fi

terraform import jira_project.example "$1"
