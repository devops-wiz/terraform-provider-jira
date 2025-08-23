// SPDX-License-Identifier: MPL-2.0

package provider

const hierarchyDescription = `
The level of the work type in the Jira issue type hierarchy:
  - -1: Sub-task (child issue type)
  - 0: Standard issue type (default level)
  - 1: Epic (Epic level)
Higher levels (2+) are available only in Jira Software Premium via Advanced Roadmaps custom hierarchy. Standard editions do not support setting levels above 0 (except -1 for sub-tasks).
References:
- Atlassian: Issue type hierarchy — https://support.atlassian.com/jira-software-cloud/docs/issue-type-hierarchy/
- Atlassian: Configure issue type hierarchy (Advanced Roadmaps) — https://support.atlassian.com/jira-software-cloud/docs/configure-issue-type-hierarchy/
`
