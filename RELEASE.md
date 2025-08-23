# Release Process for terraform-provider-jira

This document defines a repeatable, secure release process for the Terraform Provider for Jira. It aligns with internal guidelines and CI expectations. The process covers versioning (SemVer), tagging, building and signing artifacts with GoReleaser, and publishing to GitHub Releases and the Terraform Registry.

Provider address: `registry.terraform.io/devops-wiz/jira`.

## 1) Versioning Policy (SemVer)

We use Semantic Versioning (https://semver.org/):
- MAJOR (X.y.z): backwards-incompatible changes in provider behavior or schemas.
- MINOR (x.Y.z): backwards-compatible feature additions and enhancements.
- PATCH (x.y.Z): backwards-compatible bug fixes, doc updates, and non-functional changes.

Pre-releases: append a pre-release tag (e.g., `-rc.1`, `-beta.1`) for candidates. Pre-releases are published to GitHub Releases but should only be promoted to Registry after validation.

Criteria examples:
- Major: removing/renaming attributes without proper state migration, changing resource semantics in incompatible ways.
- Minor: new resources/data sources/attributes (additive), expanded filters, new docs.
- Patch: bug fixes, diagnostics improvements, retry tuning, non-breaking schema description updates.

## 2) Provenance and Metadata

Version and commit metadata are embedded into the binary via ldflags configured in `.goreleaser.yml`:
- `-X main.version={{ .Version }}`
- `-X main.commit={{ .Commit }}`

`main.go` uses these values to compose a version string and for the User-Agent. This allows support to identify exact builds.

Verify locally:
- Run a snapshot build: `task release:snapshot` and inspect the produced binary name (contains version).
- Confirm ldflags were applied: `strings ./dist/*/terraform-provider-jira_* | grep -E "\bcommit\.|\bdevops-wiz/terraform-provider-jira|version"` (optional).

## 3) Security, Signing, and Integrity

Artifacts are checksummed and signed:
- SHA256 sums are produced as `<project>_<version>_SHA256SUMS`.
- GPG signature is created for the checksum file when `GPG_FINGERPRINT` is set in the environment (see `.goreleaser.yml`).

To verify locally:
- `sha256sum -c <project>_<version>_SHA256SUMS`
- `gpg --verify <project>_<version>_SHA256SUMS.sig <project>_<version>_SHA256SUMS`

Note: Never commit or log secrets/tokens. See `.junie/guidelines.md` for redaction and security requirements.

## 4) Release Checklists

Before tagging (PR checklist):
- [ ] All CI jobs green (lint, build, unit tests, docs sync).
- [ ] Acceptance tests passed (can run locally with `.env`): `task test ACC=true`.
- [ ] CHANGELOG.md updated with entries for the release. Copy the Release Drafter draft notes into a new section:

  ## vX.Y.Z - YYYY-MM-DD

  and ensure categories follow Keep a Changelog.
- [ ] Examples and docs are up-to-date: `task gen` and commit any `docs/` changes if schema changed.
- [ ] Dependency hygiene: `go mod tidy` clean.
- [ ] Tooling versions aligned: Go matches go.mod (1.24.x) and Terraform CLI matches the CI matrix. Verify with `go version` and `terraform -version`; consider using `tfenv` to select a supported Terraform version to avoid surprises.

Local validation (optional but recommended):
- [ ] Dry-run artifacts: `task release:snapshot` (no publish) — validates ldflags, archives, checksums, manifest.

Tagging and pushing:
- Choose the next version (e.g., `vX.Y.Z`).
- Create an annotated tag and push:
  - `git tag -a vX.Y.Z -m "release vX.Y.Z"`
  - `git push origin vX.Y.Z`

Publishing (CI-driven):
- GitHub Actions release workflow (tag push) runs GoReleaser, building multi-OS/arch artifacts, checksums, and attaching `terraform-registry-manifest.json`.
- Review the GitHub Release draft, ensure notes match CHANGELOG entries.

Terraform Registry:
- The Registry pulls from GitHub Releases using the attached `terraform-registry-manifest.json` and signed checksums.
- Ensure provider address in `main.go` is `registry.terraform.io/devops-wiz/jira` (already configured).

Manual/Local build (no publish):
- `task release:local` builds artifacts locally (snapshot) for inspection.

## 5) Post-Release
- Verify installation via Terraform by pinning the new version in a sample config and running `terraform init`.
- Monitor error reports and CI for any regressions.
- Start next iteration: consider opening a milestone and moving completed tasks to [done] in `docs-internal/tasks.md`.

## 6) Troubleshooting
- GPG signing: set `GPG_FINGERPRINT` in the environment used by GoReleaser (CI Secret). For local tests, ensure your key is available and trusted.
- Docs diffs in CI: run `task gen` locally and commit generated files under `docs/`.
- Acceptance tests slow or rate-limited: see docs/index.md troubleshooting and provider retry/timeouts examples under `examples/provider/`.

## 7) Commands Reference (Taskfile)
- Generate docs: `task gen` or `task docs`
- Lint: `task lint`
- Tests (unit only default): `task test`
- Acceptance only: `task test:acc`
- All tests: `task test:all`
- Snapshot artifacts: `task release:snapshot`
- Local build: `task release:local`

## 8) Scope and Ownership
- Keep release-related changes scoped. Avoid drive-by refactors during release prep.
- Ensure security posture: no secrets in logs, redact sensitive fields in any diagnostics.
