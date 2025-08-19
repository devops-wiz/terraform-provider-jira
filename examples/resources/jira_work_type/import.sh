#!/usr/bin/env bash
# Import a Jira work type by its canonical ID.
# Usage:
#   terraform init    # ensure provider is initialized
#   terraform import jira_work_type.example <WORK_TYPE_ID>
# Example:
#   terraform import jira_work_type.example 10000

set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "Usage: $0 <WORK_TYPE_ID>" >&2
  exit 1
fi

terraform import jira_work_type.example "$1"
