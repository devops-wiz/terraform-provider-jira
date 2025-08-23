# Improvement Tasks Checklist

Note: This is a living checklist for the in-progress Terraform Provider for Jira. Each item is actionable and ordered from foundational hygiene to advanced enhancements.

## Status annotations (legend)
Use lightweight tags to reflect partial progress without changing checkbox semantics:
- [todo] not started
- [ip] in progress
- [blocked: <reason>] blocked with a short reason
- [done] completed sub-point (parent stays unchecked until all sub-points are [done]/[n/a])
- [n/a] not applicable
Optional metadata (attach to task line or a sub-point):
- [owner:@handle] assignee
- [due:YYYY-MM-DD] target date
- [progress:x/y] quick roll-up for sub-points
- [link:<url-or-#anchor>] reference to PR/issue/doc

## Example usage (not part of the checklist)
- 12. [done] Add pagination and server-side filtering support where applicable in list/data source operations: fetch complete results efficiently at scale. [owner:@alice] [link:#pagination-notes]
- 37. [ip] Establish RELEASE.md outlining tagging, changelog update, and publish steps (Registry, GitHub Releases): document repeatable release process. [owner:@maintainer] [due:2025-09-05] [progress:1/3]
    - [done] Define SemVer policy and versioning scheme; include guidance for pre-releases and patch/minor/major criteria.
    - [ip] Include checklists for tagging, running GoReleaser, and publishing to the Registry and GitHub Releases.
    - [todo] Reference where version/commit metadata is injected and how to verify artifact integrity.

---

## Tasks Overview

Below, the checklist is maintained as a flat list of tasks (no commit-series grouping) to support a one-task-at-a-time workflow.
Alignment and numbering guidance:
- Preserve original numeric IDs for checklist items; do not renumber existing items even if they move or are regrouped in this flat list.
- When moving an item, keep its original number and update any cross-references with [link:#anchor] as needed.
- When splitting an item, use sub-points; the parent remains unchecked until all sub-points are [done] or [n/a].
- Add new items by appending the next unused number; do not reuse retired numbers.
- Historical context from prior grouping is retained via per-item [series: …] tags.

### Completed Tasks

1. [done] Replace package-scope mutable env variables in provider.go with request-scoped configuration: remove globals to prevent data races and cross-instance leakage. [series: Commit 1 - Provider core configuration and auth]
2. [done] Model environment variable defaults using Terraform Plugin Framework defaults (e.g., stringdefault.FromEnvVar) in provider schema: surface env-var defaults declaratively in schema and docs. (Note: Provider-level typed defaults are not available in terraform-plugin-framework v1.15.1; env var usage is surfaced in attribute descriptions and values are resolved via centralized env-aware configuration.) [series: Commit 1 - Provider core configuration and auth]
3. [done] Centralize authentication config logic and remove duplication; clearly separate API token vs basic auth paths: one resolver validates inputs and builds the client auth. [series: Commit 1 - Provider core configuration and auth]
4. [done] Introduce a provider-level option for HTTP client timeouts and sane defaults (e.g., 30s), and wire into go-atlassian client: avoid hanging requests and allow user override. [series: Commit 1 - Provider core configuration and auth]
5. [done] Add simple retry/backoff policy for transient 429/5xx responses (configurable), preferably via go-atlassian hooks or custom http.RoundTripper: improve resilience under rate limits/outages. [series: Commit 1 - Provider core configuration and auth]
6. [done] Ensure provider address and tfplugindocs `-provider-name` are consistent; remove TODOs in main.go: align registry address and docs tooling. [series: Commit 1 - Provider core configuration and auth]
7. [done] Configure HTTP User-Agent to include version (e.g., devops-wiz/terraform-provider-jira/<version>): aid telemetry/support with versioned UA. [series: Commit 1 - Provider core configuration and auth]
8. [done] Add context propagation and timeouts to all client calls; avoid using context.Background within library operations: respect cancellation and deadlines. [series: Commit 1 - Provider core configuration and auth]

9. [done] Unify API response handling across resources/data sources using the generic helpers in interfaces.go (extend helpers as needed): consistent status checks and error diagnostics. (Validated 2025-08-15: helpers in active use via interfaces.go and response_helpers.go; keep for ongoing audit in future additions.) [series: Commit 2 - Response helpers and HTTP status consistency]
10. [done] Normalize usage of response fields (apiResp.StatusCode vs apiResp.Code) in one place; extend helpers to interpret go-atlassian ResponseScheme consistently: single accessor for HTTP status mapping. (Validated 2025-08-15: centralized via HTTPStatus/EnsureSuccessOrDiag; no direct Code/StatusCode usages remain.) [series: Commit 2 - Response helpers and HTTP status consistency]

11. [done] Make data source work_types output schema consistent: avoid Required attributes inside a Computed map; mark nested attributes as Computed: align with Terraform semantics. [series: Commit 3 - Data source semantics and timeouts]
12. [done] Decide and document keying for work_types map (e.g., key by ID for stability); implement and test: prefer stable IDs to avoid diffs on rename. [series: Commit 3 - Data source semantics and timeouts]
13. [done] Ensure deterministic ordering in data source outputs (sort keys) to avoid spurious plan diffs: stable serialization across runs. [series: Commit 3 - Data source semantics and timeouts]
14. [done] Add pagination and server-side filtering support where applicable in list/data source operations: fetch complete results efficiently at scale. [series: Commit 3 - Data source semantics and timeouts]
15. [done] Add provider and resource-level timeouts schema blocks (create/read/update/delete) where operations may be long-running: configurable per-op deadlines. [series: Commit 3 - Data source semantics and timeouts]

16. [done] Refactor workflow_status_resource to use generic helpers (Create/Read/Update/Delete) where possible to reduce bespoke error handling: minimize special cases and repeated code. [series: Commit 4 - Workflow status resource refactor and examples]
17. [done] Add examples for all resources and data sources under examples/, including minimal and advanced usage: improve discoverability and docs. [series: Commit 4 - Workflow status resource refactor and examples]

18. [done] Improve diagnostics messages: use path-based AddAttributeError for all missing/invalid config cases; ensure messages are actionable: point users at exact fields. [series: Commit 5 - Diagnostics and testing]
19. [done] Add unit tests for provider ValidateConfig and Configure paths (both API token and basic), including env-var default cases: verify core setup logic without network. [series: Commit 5 - Diagnostics and testing]
20. [done] Add unit tests for generic helpers (interfaces.go) with fake client/responses to validate status-code/error handling: ensure shared code behaves correctly. [series: Commit 5 - Diagnostics and testing]
21. [done] Expand acceptance tests to cover negative scenarios and import functionality for all resources: validate real-world flows and import support. [series: Commit 5 - Diagnostics and testing]
22. [done] Add acceptance tests for data source filtering by ids and names, including mixed-case and not-found behavior: confirm filtering semantics and edge cases. [series: Commit 5 - Diagnostics and testing]
23. [done] Implement and document a test sweeper for temporary resources created during acceptance tests: avoid leftover test artifacts in Jira. [series: Commit 5 - Diagnostics and testing]

24. [done] Add .golangci.yml configuration aligned with guidelines; enable key linters (errcheck, staticcheck, govet, ineffassign, unused, misspell, gofmt, goimports, unparam, unconvert, prealloc, makezero, nilerr, durationcheck, copyloopvar): enforce code quality locally/CI. [series: Commit 6 - CI, linting, and generated docs]
25. [done] Integrate golangci-lint into CI workflow; ensure `golangci-lint run` passes locally and in CI: keep linting enforced on PRs. [series: Commit 6 - CI, linting, and generated docs]
26. [done] Add CI step to run `go generate ./...` and fail on git diff for docs (tfplugindocs) to keep docs in sync: prevent stale generated docs. [series: Commit 6 - CI, linting, and generated docs]
27. [done] Add Terraform CLI version matrix to CI for acceptance tests (ensure secrets gating); align with .github/workflows/test.yml: verify compatibility across TF versions. [series: Commit 6 - CI, linting, and generated docs]
28. [done] Add Go race detector and coverage runs to CI for unit tests (`-race`, `-cover`): catch concurrency issues and track coverage. [series: Commit 6 - CI, linting, and generated docs]
29. [done] Generate and commit docs via `go generate ./...`; verify docs/data-sources and docs/resources pages are present and accurate: keep registry docs current. [series: Commit 6 - CI, linting, and generated docs]

30. [done] Standardize Go naming for initialisms (Id → ID, Api → API) in internal structs and functions; keep tfsdk tags stable: follow Go style without breaking state. [series: Commit 7 - Docs and contributor experience]
31. [done] Audit and add missing MarkdownDescription vs Description consistently across schemas: richer generated docs where helpful. [series: Commit 7 - Docs and contributor experience]
32. [done] Expand docs/index.md with troubleshooting, rate limits, and auth method guidance; link to environment variable usage: user-facing guidance and FAQs. [series: Commit 7 - Docs and contributor experience]
33. [done] Create CONTRIBUTING updates to reflect linting/testing flows, Taskfile targets, and acceptance testing requirements: onboard contributors effectively. [series: Commit 7 - Docs and contributor experience]
34. [done] Add CODEOWNERS and PR template to standardize reviews and contributions: clarify ownership and PR expectations. [series: Commit 7 - Docs and contributor experience]

35. [done] Introduce GoReleaser config to inject version/commit into main.version and produce release artifacts: reproducible, versioned builds. [series: Commit 8 - Release and change history]
36. [done] Establish RELEASE.md outlining tagging, changelog update, and publish steps (Registry, GitHub Releases): document repeatable release process. [progress:3/3] [series: Commit 8 - Release and change history]
    - [done] Define SemVer policy and versioning scheme; include guidance for pre-releases and patch/minor/major criteria.
    - [done] Include checklists for tagging, running GoReleaser, and publishing to the Registry and GitHub Releases.
    - [done] Reference where version/commit metadata is injected and how to verify artifact integrity.
37. [done] Add CHANGELOG entry templates and automate via GitHub Actions or Keep a Changelog format: maintain transparent change history. [progress:2/2] [series: Commit 8 - Release and change history]
    - [done] Decide on automation approach (Keep a Changelog, Release Drafter, or custom GH Action).
    - [done] Document the workflow for updating CHANGELOG on each release; ensure CI validates CHANGELOG presence on tags.

38. [done] Review and redact any sensitive values from errors/diagnostics; never print tokens/usernames in logs: strengthen security posture. [series: Commit 9 - Security hardening]

39. [done] Add provider-level configuration validation for mutually exclusive attributes using framework validators where possible: declarative conflicts/requirements. [series: Future Commit A - Provider validators and config polish]
40. [done] Canonicalize env vars and document aliases: [progress:3/3] [series: Future Commit A - Provider validators and config polish]
    - [done] Canonical: JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN.
    - [done] Aliases (lower precedence): JIRA_BASE_URL, JIRA_EMAIL. Document precedence and update schema descriptions.
    - [done] Support alias reading in configuration logic with canonical > alias precedence.
41. [done] Align provider attribute names and timeouts: [progress:2/2] [series: Future Commit A - Provider validators and config polish]
    - [done] Use endpoint and api_auth_email; single http_timeout_seconds knob confirmed.
    - [done] Update docs and examples accordingly.
42. [done] Remove provider-level “premium” flag from schema/config for now; keep as future consideration. [progress:2/2] [series: Future Commit A - Provider validators and config polish]
    - [done] Update configuration, docs, and tests to remove/ignore premium.
    - [done] Add a note in roadmap to re-evaluate implementing a premium flag later.

43. [done] Replace magic numbers/status codes in code with shared constants/enums and document expected API responses: improve readability and maintainability. [progress:3/3] [series: Future Commit B - HTTP constants, telemetry, and troubleshooting docs]
    - [done] Define a shared set of HTTP status constants and helper functions for response interpretation.
    - [done] Replace remaining ad-hoc status checks in code with the shared helpers.
    - [done] Document expected success/error responses per resource/data source and link from docs/troubleshooting.
44. [done] Add telemetry/debug toggle (e.g., TF_LOG or provider flag) to increase logging verbosity safely: enable opt-in troubleshooting without leaking secrets. [progress:3/3] [series: Future Commit B - HTTP constants, telemetry, and troubleshooting docs]
    - [done] Honor TF_LOG while providing a provider-level debug flag for structured diagnostic logs.
    - [done] Ensure redaction of sensitive values in all debug paths; add unit tests for redaction helpers.
    - [done] Document how to enable debug logs safely and what information is surfaced.
45. [done] Expand Troubleshooting with rate-limit guidance and retry behavior: help users operate under 429/5xx conditions. [progress:4/4] [series: Future Commit B - HTTP constants, telemetry, and troubleshooting docs]
    - [done] Document how retries/backoff work, Retry-After handling, and recommended mitigations.
    - [done] Include guidance for tuning provider retry settings and concurrency if applicable.
    - [done] Provide sample troubleshooting logs (429 with Retry-After, 429 without header, transient 5xx, concurrency saturation).
    - [done] Add example TF_LOG=DEBUG snippets and notes on -parallelism and provider retry knobs.

46. [done] Review license headers and ensure MPL-2.0 is consistently applied; add copywrite validation in CI: automate license compliance. [progress:3/3] [series: Future Commit C - Compliance, pre-commit, and repo hygiene]
    - [done] Integrate license header check into CI and pre-commit (copywrite).
    - [done] Add/fix headers across source files; exclude generated files as appropriate.
    - [done] Fail CI when headers are missing or incorrect.
47. [done] Ensure pre-commit hooks mirror CI checks; document how to run `pre-commit run -a`: catch issues before pushing. [progress:3/3] [series: Future Commit C - Compliance, pre-commit, and repo hygiene]
    - [done] Mirror: golangci-lint, goimports/gofmt, docs generation diff check, license header check, and tests (fast path).
    - [done] Add terraform fmt for example HCL where applicable.
    - [done] Document exact commands and expected local setup in CONTRIBUTING.
48. [done] Add scratch/ examples cleanup: ensure scratch files are ignored in linters and not shipped: keep CI/release artifacts clean. [progress:0/3] [series: Future Commit C - Compliance, pre-commit, and repo hygiene]
    - [done] Add/confirm .gitignore patterns and linter exclusions for scratch/ and similar paths.
    - [done] Ensure .goreleaser excludes scratch/ and non-distribution files from artifacts.
    - [done] Validate examples are formatted, linted, and included intentionally.

49. [done] Configuration & Auth (FR-1): implement env precedence (canonical > alias), sensitive handling, required-with/mutual exclusivity validators; expose http_timeout and retry knobs; respect_rate_limits default; User-Agent includes version. Acceptance: unit tests for precedence and validators; provider docs updated; docs generate with no diff in CI. [series: FR/NFR Alignment and Provider Validators]
50. [done] Client Wrapper & Helpers (FR-2): centralize retries/backoff honoring Retry-After, HTTP status mapping, model→state conversions, narrow interfaces for fakes. Acceptance: unit tests for retry mapping/redaction; no bespoke per-resource retries; shared helpers used in new code. [series: FR/NFR Alignment and Provider Validators]
51. [done] CRUD Coverage (FR-3): resources call go-atlassian services with clear diagnostics; import by canonical IDs. Acceptance: CRUD + import tests green in CI. [series: FR/NFR Alignment and Provider Validators]
52. [done] Data Sources (FR-4): implement list/lookup with name/ID filters, mutual exclusivity validators, pagination/server-side filtering. Acceptance: list/lookup tests verify behavior and ordering. [series: FR/NFR Alignment and Provider Validators]
53. [done] Idempotence & State (FR-5): persist canonical IDs; normalize values; deterministic ordering. Acceptance: unit tests confirm stable state and ordering across runs. [series: FR/NFR Alignment and Provider Validators]
54. [done] Diagnostics & 429 Handling (FR-6): map Atlassian errors to actionable diagnostics; 404 disappearance; 429 guidance and retries. Acceptance: simulated response tests; troubleshooting docs updated. [series: FR/NFR Alignment and Provider Validators]
55. [done] Docs & Examples (FR-7): tfplugindocs covers every public item with examples, including import ID formats and Premium notes. Acceptance: `go generate ./...` produces no diff in CI. [series: FR/NFR Alignment and Provider Validators]
56. [done] Tests (FR-8): unit + TF_ACC acceptance tests (CRUD + import + negative/permissions), sweeper, env var docs. Acceptance: CI green; scheduled acceptance runs optional with secrets. [series: FR/NFR Alignment and Provider Validators]

57. [done] jira_workflow_status — CRUD + import; acceptance tests; docs/examples; troubleshooting updated; import format documented. Acceptance: resource + import tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
58. [done] jira_work_type — CRUD + import (standard/subtask); acceptance tests; docs/examples; import format documented. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]

### Backlog / In Progress

1. [done] jira_project — CRUD + import; acceptance tests; docs/examples; required permissions documented. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
2. [done] jira_project — lookup by key/ID. [progress:2/2] Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
3. [done] jira_projects — list/filter; pagination. Acceptance: tests; docs/examples; no-diff. [progress:3/3] [series: MVP Data Sources Implementation (Plan §18)]
4. [ip] [progress:1/2] jira_project_category resource and data-source
5. [todo] jira_fields — list/filter; deterministic ordering; server-side filtering if supported. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
6. [todo] jira_priority — CRUD + import (custom priorities); acceptance tests; docs/examples. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
7. [todo] jira_priorities — list; deterministic ordering. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
8. [todo] jira_status_categories — list; deterministic ordering. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
9. [todo] jira_resolution — CRUD + import (custom resolutions); acceptance tests; docs/examples. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
10. [todo] jira_resolutions — list; deterministic ordering. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
11. [todo] jira_project_component — CRUD + import; acceptance tests; docs/examples; project key/ID handling. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
12. [todo] jira_workflow_statuses — list/lookup; pagination; deterministic ordering. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
13. [todo] jira_workflow — CRUD + import; statuses/basic transitions; initial validators/conditions; advanced/unsupported read-only with warnings; no post-functions. Acceptance: tests pass; docs call out read-only parts and warnings; import format documented. [series: MVP Resources Implementation (Plan §18)]
    - [todo] API surface review and client helpers. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/87]
    - [todo] Define resource schema and state model. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/88]
    - [todo] Implement Create operation. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/89]
    - [todo] Implement Read operation (flatten). [link:https://github.com/devops-wiz/terraform-provider-jira/issues/90]
    - [todo] Implement Update operation. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/91]
    - [todo] Implement Delete operation. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/92]
    - [todo] Import support and ID format docs. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/93]
    - [todo] Minimal transitions + validators/conditions read-only with warnings. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/94]
    - [todo] Acceptance tests (CRUD + import + negatives). [link:https://github.com/devops-wiz/terraform-provider-jira/issues/95]
    - [todo] Docs, examples, and troubleshooting. [link:https://github.com/devops-wiz/terraform-provider-jira/issues/96]
14. [todo] jira_workflow (lookup) — lookup by name/ID; base fields; deterministic outputs. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
15. [todo] jira_work_types — list/lookup by ids/names; conflict validation; pagination; deterministic ordering. Acceptance: unit + acceptance tests; docs/examples; tfplugindocs no-diff. [series: MVP Data Sources Implementation (Plan §18)]
16. [todo] jira_issue_link_type — CRUD + import; acceptance tests; docs/examples. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
17. [todo] jira_issue_link_types — list; deterministic ordering. Acceptance: tests; docs/examples; no-diff. [series: MVP Data Sources Implementation (Plan §18)]
18. [todo] jira_project_version — CRUD + import; acceptance tests; docs/examples. Acceptance: tests pass; docs no-diff. [series: MVP Resources Implementation (Plan §18)]
19. [todo] Attributes: include_subtasks, top_level_only, allowed_levels, hierarchy_min, hierarchy_max, require_consistent_hierarchy; import docs and examples. Acceptance: schema and docs updated; examples added. [series: Hierarchy-aware jira_work_types Data Source]
20. [todo] Validators: mutual exclusivity (include_subtasks vs top_level_only), bounds (min ≤ max), path-based diagnostics. Acceptance: unit tests for validators; diagnostics verified. [series: Hierarchy-aware jira_work_types Data Source]
21. [todo] Behavior: client-side filtering by HierarchyLevel; warnings when filtered; error on mixed-levels when consistency required; deterministic outputs. Acceptance: unit + acceptance tests verify filtering/warnings/errors and ordering. [series: Hierarchy-aware jira_work_types Data Source]
22. [todo] Tests: unit and acceptance for all new attributes and cases (including mixed levels). Acceptance: tests pass in CI. [series: Hierarchy-aware jira_work_types Data Source]
23. [todo] Docs: tfplugindocs regenerated; examples updated; CI no-diff. Acceptance: docs job passes with no diff. [series: Hierarchy-aware jira_work_types Data Source]

24. [todo] Release Drafter with canonical labels and Keep a Changelog template. Acceptance: workflow present; PRs labeled; draft release shows categorized notes. [series: CI/CD, Release, and Supply Chain Guardrails]
25. [todo] PR label guard enforcing exactly one canonical category or skip label. Acceptance: CI fails when multiple/none labels set; remediation guidance in CONTRIBUTING. [series: CI/CD, Release, and Supply Chain Guardrails]
26. [todo] Changelog validator on tags with remediation guidance. Acceptance: tag workflow fails if CHANGELOG not updated; guidance linked. [series: CI/CD, Release, and Supply Chain Guardrails]
27. [todo] Docs no-diff gate: run `go generate ./...` and fail on diff. Acceptance: CI job exists and fails on violation. [series: CI/CD, Release, and Supply Chain Guardrails]
28. [todo] Signed commits/tags; verify in CI. Acceptance: CI validates signatures on release tags/commits. [series: CI/CD, Release, and Supply Chain Guardrails]
29. [todo] GoReleaser: checksums/signatures; SBOM and provenance; multi-arch artifacts. Acceptance: release artifacts include signatures, checksums, SBOM, provenance. [series: CI/CD, Release, and Supply Chain Guardrails]
30. [todo] Secret scanning in CI and pre-commit. Acceptance: scans run and fail on findings; exemptions documented. [series: CI/CD, Release, and Supply Chain Guardrails]
31. [todo] Vulnerability scans (govulncheck/OSV). Acceptance: CI job runs and fails on high/critical; suppression workflow documented. [series: CI/CD, Release, and Supply Chain Guardrails]
32. [todo] Dependency automation (Renovate/Dependabot). Acceptance: bot PRs open; config present; schedule documented. [series: CI/CD, Release, and Supply Chain Guardrails]
33. [todo] go.mod/go.sum tidy checks. Acceptance: CI fails on dirty modules; Taskfile has tidy target. [series: CI/CD, Release, and Supply Chain Guardrails]
34. [todo] Governance/access controls: CODEOWNERS and branch protections; approvals for CI/release/security; avoid self-approval. Acceptance: policies documented and enforced. [series: CI/CD, Release, and Supply Chain Guardrails]
35. [todo] Toolchain alignment: Go >= 1.24, Terraform Core >= 1.6; pinned TPF/go-atlassian/tfplugindocs; Task runner integration. Acceptance: go.mod/tool versions pinned; CI matrix updated. [series: CI/CD, Release, and Supply Chain Guardrails]

36. [todo] Schemes (workflow/issue type/screen): implement resources and associations (e.g., workflow_scheme, issue_type_scheme, issue_type_scheme_association, screen, screen_scheme) with CRUD + import and deterministic state/order. Acceptance: CRUD + import tests pass; docs/examples updated; tfplugindocs no-diff. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
37. [todo] Roles & actors: project_role and project_role_actors resources; role creation then actor grants ordering. Acceptance: tests assert roles before grants; docs note dependency guidance. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
38. [todo] Security/notification/permission schemes: resources for permission_scheme, notification_scheme, issue_security_scheme; association resources to projects where applicable. Acceptance: CRUD + import tests; docs detail import formats and permissions. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
39. [todo] Webhooks: resource for webhook with secret redaction and test fakes; idempotent updates. Acceptance: unit + acceptance tests; docs include security notes. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
40. [todo] Project category: resource + data source; import by ID; deterministic ordering in lists. Acceptance: tests pass; docs/examples updated. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
41. [todo] Audit/time tracking settings: resources/data sources where supported; read/update with safe defaults. Acceptance: tests cover read/update; docs explain plan-only vs apply behaviors. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
42. [todo] Premium hierarchy resources: model read-only vs writable surfaces; warn for unsupported advanced features; include examples. Acceptance: docs with warnings; tests for read behaviors. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
43. [todo] Ordering/dependency guidance: document and enforce sequencing (project features before components/versions; roles before grants; identity lookups before refs; field configs before attachments; security scheme before level members). Acceptance: docs updated; plan modifiers/validators enforce where possible; unit tests validate ordering. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
44. [todo] Import formats: define and document import ID formats for all new resources and associations. Acceptance: import docs present; import tests green. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]
45. [todo] Acceptance templates: add focused tf.tmpl per resource to validate idempotency and cleanup. Acceptance: templates committed; tests reference templates. [series: v1.0 breadth — Schemes & Admin Resources (Plan §18 Phase 4)]

46. [todo] Publish control mappings (SOX/SOC 2/HIPAA) and evidence links. Acceptance: docs page exists; links validated in CI link-check. [series: Enterprise Policy Mapping & Continuous Controls Validation]
47. [todo] LTS/support/deprecation policy: document supported versions and policies. Acceptance: policy page updated; referenced from README. [series: Enterprise Policy Mapping & Continuous Controls Validation]
48. [todo] Audit artifacts retention: define retention for CI artifacts (coverage, flakiness, 429 metrics) and export schedule. Acceptance: CI uploads artifacts; retention configured and documented. [series: Enterprise Policy Mapping & Continuous Controls Validation]
49. [todo] Continuous controls validation: scheduled workflow validates branch protections, signing, scans, Actions hardening; opens issues on drift. Acceptance: scheduled job present; fails on drift with actionable errors. [series: Enterprise Policy Mapping & Continuous Controls Validation]
50. [todo] Enterprise consumption templates: provide organization policy templates for provider usage. Acceptance: templates added under docs-internal or .github; referenced in docs. [series: Enterprise Policy Mapping & Continuous Controls Validation]

51. [todo] README: provider configuration, least-privilege token guidance, TF_ACC envs (JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN; optional JIRA_USERNAME, JIRA_PASSWORD), troubleshooting (401/403, 404, 429 with tuning and -parallelism, TLS/endpoint). Acceptance: README updated; examples runnable. [series: Documentation & Troubleshooting]
52. [todo] Docs: import ID formats for each resource/data source; Premium feature notes where relevant. Acceptance: docs pages updated; tfplugindocs no-diff. [series: Documentation & Troubleshooting]
53. [todo] Examples: runnable configs for each resource and data source; negative/permissions examples where safe. Acceptance: examples validated by tests or manual check. [series: Documentation & Troubleshooting]
54. [todo] Cross-links: link troubleshooting, guidelines, and requirements sections using anchors. Acceptance: links valid. [series: Documentation & Troubleshooting]

55. [todo] Structured, redacted debug logs gated by TF_LOG and provider toggle; request-scoped fields. Acceptance: redaction unit tests exist; manual smoke checks confirm no secrets. [series: Observability & Logging]
56. [todo] Metrics/telemetry artifacts in CI: track unit/acceptance coverage, roadmap coverage, acceptance flakiness, and rate-limit incidents as CI artifacts. Acceptance: artifacts published and trends observable. [series: Observability & Logging]
