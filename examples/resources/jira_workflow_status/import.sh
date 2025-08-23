#!/usr/bin/env bash
# Import a Jira workflow status by its canonical ID.
# Usage:
#   terraform init    # ensure provider is initialized
#   terraform import jira_workflow_status.example <STATUS_ID>
# Example:
#   terraform import jira_workflow_status.example 10001

set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "Usage: $0 <STATUS_ID>" >&2
  exit 1
fi

terraform import jira_workflow_status.example "$1"
