<!--
Title format suggestion: feat|fix|docs|refactor|test|chore(scope): short summary
Example: feat(provider): add configurable HTTP timeouts
-->

## Summary

<!-- Briefly describe the change. What does this PR do? -->

## Motivation & Context

<!-- Link related issues and context. Use Closes/Fixes for auto-close. -->
- Closes #
- Relates to #

## Changes

<!-- High-level list of changes. Keep concise. -->
-

## Docs

<!-- If schema or behavior changed, ensure docs are regenerated and committed. -->
- Updated docs via Taskfile (see Checklist)
- Examples updated (if applicable)

## Testing

<!-- Outline how you tested locally. Include commands and expected results. Prefer Taskfile targets. -->
- Unit tests: `task test` (or `task test:unit`)
- Acceptance tests: `task test:acc` (requires .env with JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN)
- Additional notes:

## Screenshots / Logs (redacted)

<!-- Include relevant output, ensuring credentials/tokens are NOT present. -->

## Backward Compatibility

<!-- Any breaking changes to provider config, resource schema, or behavior? If yes, document migration steps. -->
- Breaking changes: Yes/No
- Migration notes (if any):

## RELEASE (Changelog)

<!--
Required for user-facing changes. This section is consumed by release tooling.
If there are no user-facing changes, set the `skip-release-notes` label and leave this section empty.
Categories follow Keep a Changelog: added | changed | deprecated | removed | fixed | security
-->

- Category: added | changed | deprecated | removed | fixed | security
- Release note (concise, user-facing):
  -
- Breaking change: Yes/No
- Upgrade notes (if breaking):

## Checklist (prioritized)

### Critical before merge
- [ ] Acceptance tests executed and passed: `task test:acc` (Task loads .env) — if applicable; docs/CI-only PRs may skip.
- [ ] Changelog provided:
  - [ ] RELEASE section above filled out for user-facing changes
  - [ ] OR `skip-release-notes` label applied when there are no user-facing changes
- [ ] Docs generated and committed:
  - [ ] `task gen` (runs `go generate ./...`) and/or `task docs` (tfplugindocs)
  - [ ] No remaining git diff under `docs/`
  - [ ] Examples changed: ran `task gen` (formats examples and validates/generates docs)
- [ ] Appropriate labels applied (exactly one canonical category label `type:*` OR `skip-release-notes`; enforced by PR Label Guard)
- [ ] No secrets in logs/errors; sensitive values redacted

### Standard checks
- [ ] Imports organized and code formatted (Taskfile): `task goimports` and `task fmt`
- [ ] Lint passes locally: `task lint`
- [ ] Build succeeds: `task build`
- [ ] Unit tests pass: `task test` (or `task test:unit`)
- [ ] Examples/README/docs updated if schema/API changed

### Scope gating (optional)
- [ ] Docs/CI-only: this PR changes only docs/CI/chore files (no `**/*.go`, `go.mod`, `go.sum`, `internal/**`, `main.go`, or `templates/**/*.md.tmpl`; Taskfile behavior unchanged). Safe to skip Go/TFPF checks; maintainers may still request full runs.

## Labels

<!-- Apply labels to help triage and release notes. -->
- Type (choose one): `type:feat` | `type:fix` | `type:docs` | `type:refactor` | `type:test` | `type:chore`
- Area (as applicable): `area:provider` | `area:docs` | `area:resource:<name>` | `area:data-source:<name>`
- Release notes: exactly one of `release-note` (user-facing) OR `skip-release-notes` (no user-facing change)
- Breaking change: add `breaking-change` (and ensure upgrade notes above)
- If unsure, add `triage:needs-category` and a maintainer will adjust.

## Reviewer Notes

<!-- Call out areas that need extra attention, trade-offs made, or follow-ups planned. -->
