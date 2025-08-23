# Contributing to terraform-provider-jira

Thanks for taking the time to contribute!

This document outlines how to set up your environment, build, test, lint, and propose changes. It also links to our Code of Conduct and Security Policy.

## Toolchain

- Go 1.24.x (we test with 1.24.x in CI; go.mod sets 1.24.3)
- Terraform CLI >= 1.0 (for running examples/acceptance tests)
- Optional: [Task](https://taskfile.dev) for developer convenience (Taskfile.yml provided)
- Optional: [pre-commit](https://pre-commit.com) for local hooks

## Getting started

1. Clone this repository.
2. Ensure you have Go 1.24.x installed.
3. (Optional) Install pre-commit and set up hooks:
   ```sh
   pipx install pre-commit   # or pip install --user pre-commit
   pre-commit install
   ```
4. (Optional) Install helper tools:
   ```sh
   go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
   go install github.com/hashicorp/copywrite@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Build, Lint, Test

Using Task:

- Format: `task fmt`
- Imports + format: `task goimports`
- Lint: `task lint`
- Build (quick check): `task build`
- Install locally: `task install`
- Docs generation: `task gen` (runs `go generate ./...`) and `task docs` (runs `tfplugindocs generate`)
- Tests:
  - Unit/default: `task test`
  - Shortcuts: `task test:unit`, `task test:acc`, `task test:all`
  - Flags: `CTC=true` (clean cache), `VRBS=true` (verbose), `RACE=true`, `COVER=true COVERPROFILE=coverage.out`
  - Pattern filter: `task test RUN_PATTERN="TestAccWorkTypeResource_basic"`
- Sweeper cleanup: `task sweep` (loads `.env`)

Without Task:

```sh
go fmt ./...
# Lint with golangci-lint using repo config
golangci-lint run
# Build
go build ./...
# Tests (skip acceptance by default)
go test ./... -run TestUnit -count=0
```

## Pre-commit hooks

We provide a pre-commit configuration to mirror CI checks so you can catch issues locally before pushing.

Hooks included:
- golangci-lint (using repo .golangci.yml)
- goimports and go fmt
- terraform fmt -check -recursive for examples/
- copywrite SPDX license header check (uses copywrite.hcl)
- docs generation no-diff (runs `go generate ./...` and fails on docs/ diff)
- go test (unit only; TF_ACC disabled)

Setup:
```sh
pipx install pre-commit    # or: pip install --user pre-commit
pre-commit install
```

Usage:
```sh
# Run on all files
pre-commit run -a
# Or via Task
task precommit
```

If a hook fails:
- Follow the message (e.g., run `task gen` to regenerate docs and commit changes).
- Install missing tools as suggested (goimports, copywrite, terraform CLI).
- Re-run `pre-commit run -a` until all checks pass.

## Acceptance tests

Acceptance tests talk to a real Jira Cloud instance and require environment variables:

- `JIRA_ENDPOINT`
- `JIRA_API_EMAIL`
- `JIRA_API_TOKEN`

Task auto-loads a `.env` file at the repo root for tests. For local runs, place the above variables in `.env` and do not commit it.

Run (with Task):

```sh
task test ACC_ONLY=true
# or the shortcut
task test:acc
# filter to a specific test
task test ACC_ONLY=true RUN_PATTERN="TestAccWorkTypeResource_basic"
```

Or directly:

```sh
TF_ACC=1 go test ./internal/provider -v -run TestAcc
```

Be mindful of Atlassian rate limits and costs. Prefer running filtered subsets when validating changes. A sweeper is available via `task sweep` to clean up test artifacts.

## Documentation

We use `tfplugindocs` (wired via `go generate`) to generate docs from schemas into the `docs/` directory.

- Generate via Go: `task gen` (runs `go generate ./...`)
- Generate via tfplugindocs directly: `task docs` (runs `tfplugindocs generate`)
- CI enforcement: the CI pipeline runs `go generate ./...` and fails if there is any git diff under `docs/`. If you change schemas, run generation locally and commit updated files under `docs/`.
- Examples-only changes: still run `task gen` — this formats `examples/` (terraform fmt -recursive) and runs tfplugindocs generate/validate to catch broken snippets.

## CI Expectations

- Lint: `golangci-lint` must pass locally (Task: `task lint`) and in CI.
- Build & Unit Tests: CI builds the provider and runs unit tests (race/coverage may be enabled). Locally use `task test` and optional flags (e.g., `RACE=true`, `COVER=true`).
- Docs Sync: CI validates generated docs are up to date; ensure you ran `task gen`/`task docs` and committed changes.
- Acceptance: CI runs acceptance tests using a Terraform CLI version matrix when Jira credentials are available via repository secrets. Keep tests idempotent and clean up resources; use the sweeper if needed.

## Skipping Go/TFPF checks for docs/CI-only changes

- You can skip provider build/lint/tests and tfplugindocs when your PR only changes: Markdown docs (README/CHANGELOG/CONTRIBUTING/RELEASE), `.github/**` workflows/templates, `.junie/**`, `docs-internal/**`, or other non-code chore files.
- Do not skip if any of: `**/*.go`, `go.mod`, `go.sum`, `main.go`, `internal/**`, or `templates/**/*.md.tmpl` are modified, or when Taskfile changes affect build/test/docs generation.
- Regardless, if files under `examples/` changed, run `task gen` — this formats examples and re-generates/validates docs.

## Changelog and Release notes

We follow Keep a Changelog and Semantic Versioning. Release notes are drafted automatically using Release Drafter.

- PR labels drive changelog categories (canonical):
  - Added: `type:feat`
  - Changed: `type:change`, `type:refactor`, `type:docs`, `type:chore`, `type:test`
  - Deprecated: `type:deprecated`
  - Removed: `type:breaking`
  - Fixed: `type:fix`
  - Security: `security`
- Note: Use only the canonical `type:*` labels above. Legacy synonyms (e.g., `feature`, `enhancement`, `bug`) are not recognized by automation and will cause the PR to fail the label guard.
- To exclude a PR from release notes, apply `skip-release-notes` (or `skip-changelog`).
- For user-facing changes, ensure the PR includes a RELEASE section in the PR template; maintainers may adjust labels for correct categorization.
- On tag pushes (e.g., `v1.2.3`), CI validates that `CHANGELOG.md` contains a heading for that version; ensure you add a versioned section when cutting a release.
- When preparing a release, copy the Release Drafter draft notes into CHANGELOG.md under the new version heading.

Local verification:
- Lint: `task lint`
- Docs: `task gen` (and `task docs` if needed)
- Examples changed: `task gen` (formats examples and validates/generates docs)
- Tests: `task test` and acceptance via `task test:acc` (Task loads `.env`).

## License headers

We standardize SPDX headers. To (re)apply headers automatically:

```sh
copywrite headers
```

A Taskfile target may be provided as `task license`.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## Security

Please do not open public issues for security vulnerabilities. See our [Security Policy](SECURITY.md) for reporting instructions.

## Pull Requests

- Keep changes focused; separate refactors from features/bug fixes.
- Include tests where possible (unit and/or acceptance).
- Ensure `golangci-lint` passes locally or explain any expected findings.
- Update documentation relevant to your changes.
- Describe the motivation and context in the PR description.


## JetBrains HTTP Client environment files (git-ignored)

We ship curated HTTP requests under `docs-internal/http/` for exploring Jira REST API behavior with the JetBrains HTTP Client.

Local environment files are used to supply base URL and credentials to these requests:
- `http-client.env.json`
- `http-client.private.env.json` (recommended)

Both files are intentionally ignored by Git to prevent accidental leakage of secrets. See `.gitignore` (JetBrains HTTP Client section) which lists:
- `http-client.env.json`
- `http-client.private.env.json`

How to use locally:
1) Copy `http-client.env.json.example` at the repo root to `http-client.private.env.json` (preferred).
2) Fill the values for your test Jira Cloud site:
   - `JIRA_BASE_URL` (e.g., https://your-org.atlassian.net)
   - `JIRA_EMAIL`
   - `JIRA_API_TOKEN`
   - `JIRA_BASIC_AUTH` (base64 of "email:api-token"). See examples in the example file.
3) In your JetBrains IDE, select the environment (e.g., `dev`) in the HTTP Client toolbar and run requests from `docs-internal/http/*.http`.

Security and safety notes:
- Never commit real credentials. These files are git-ignored by default; verify with `git status` before committing.
- Prefer `http-client.private.env.json` for local-only use and keep it out of synced/shared folders.
- The IDE may keep request/response history under `.idea/httpRequests`. Clear history after testing and avoid sharing saved responses.
- Consider disabling "Save authorization headers" in the HTTP Client settings.
- If a token may have been exposed (logs, screenshots, or responses), rotate/revoke the `JIRA_API_TOKEN` immediately.
- Run destructive examples only against non-production Jira sites.

For a step-by-step guide and additional tips, see README.md → "JetBrains HTTP Client requests (API testing)".
