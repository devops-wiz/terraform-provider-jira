# Improvement Plan: Terraform Provider for Atlassian Jira Cloud

Project: terraform-provider-jira (Terraform Plugin Framework + go-atlassian)
Registry address: registry.terraform.io/devops-wiz/jira
Author: DevOps Wiz Platform Team

This improvement plan synthesizes docs-internal/requirements.md and aligns with .junie/guidelines.md. It focuses on minimal, safe, and high-impact delivery steps to meet FR-1..FR-8 and NFR-1..NFR-12, while keeping scope to planning and delivery activities (no code unless essential to clarify behavior). All actions include acceptance criteria tied to CI checks or artifacts.

## 0. Executive Summary

Goals
- Idiomatic Terraform provider using the Terraform Plugin Framework and go-atlassian for Jira Cloud.
- Accurate schemas, safe/idempotent CRUD, canonical IDs/import, and deterministic state.
- tfplugindocs-generated docs with usable examples; acceptance tests validate real server state.
- Reliable under rate limits, secure handling of secrets, structured/redacted logs, pinned dependencies.

Constraints
- Jira Cloud via go-atlassian; some endpoints require admin privileges for acceptance tests.
- Respect Atlassian rate limits; avoid N+1 patterns; prefer canonical IDs and deterministic ordering.
- CI must pass lint/unit/race/coverage, validate docs generation (no diff), and optionally run TF_ACC tests with secrets.

Approach
- Central client wrapper and uniform diagnostics.
- Unified provider config with env precedence and HTTP/retry knobs.
- MVP resource/data-source coverage first; expand breadth post-MVP.

Status (high level)
- Foundation in place (wrappers, ordering, docs pipeline). Focus next on: release automation/guardrails, schema validators for auth, provider debug toggle, supply-chain integrity (signing/SBOM/provenance), and dependency automation.

---

## 1. Architecture & Client Wrapper (FR-2)

Actions
- Maintain a shared client wrapper around go-atlassian:
  - Centralize retries/backoff (429/5xx) with capped exponential backoff and jitter; respect Retry-After.
  - Encapsulate jira.Client construction with auth, endpoint, http_timeout, retry knobs, respect_rate_limits, and custom User-Agent embedding provider version.
  - Provide shared helpers: ID parsing/import formats, error/HTTP status mapping, and model→state converters.
  - Expose narrow interfaces for used go-atlassian services to enable fakes in unit tests.

Acceptance
- Unit tests: retry/backoff policy, 429 mapping honoring Retry-After, error redaction, and interface-based injection.
- Docs: internal notes outline wrapper responsibilities; no per-resource bespoke retry logic remains.

Status
- Implemented and used; continue auditing new surfaces to use the wrapper consistently.

---

## 2. Authentication & Provider Configuration (FR-1, NFR-1)

Actions
- Provider attributes (names must match requirements.md):
  - endpoint (string)
  - api_auth_email (sensitive string), api_token (sensitive string)
  - username (sensitive string), password (sensitive string) — Basic auth optional
  - http_timeout (duration), retry_max_attempts (number), retry_initial_backoff (duration), retry_max_backoff (duration), respect_rate_limits (bool, default true)
- Environment variable precedence (canonical > alias):
  - Canonical: JIRA_ENDPOINT → endpoint; JIRA_API_EMAIL → api_auth_email; JIRA_API_TOKEN → api_token
  - Aliases (lower precedence): JIRA_BASE_URL → endpoint; JIRA_EMAIL → api_auth_email
- Validate auth configuration:
  - API token is primary; Basic optional (and mutually exclusive when both token and basic credentials are provided).
  - Mark all credentials sensitive; never log values.
  - Note: OAuth/App auth is a future migration path (not implemented yet).
- Ensure User-Agent includes provider version; propagate to HTTP client.

Acceptance
- Unit tests: ValidateConfig/Configure for env precedence, required-with/mutual exclusivity, defaults for timeout/retries, and UA includes version.
- Docs: README and provider docs list canonical envs, alias precedence, and retry knobs.

Status
- Core implemented; add explicit validators for conflicts and required-with semantics.

---

## 3. Data Model & State Mapping (FR-5, Section 11)

Actions
- Persist canonical IDs only; avoid ephemeral server fields.
- Document which nested fields are persisted vs computed/read-only.
- Normalize values and apply plan modifiers for deterministic state.
- Ensure data sources use stable keys (ID) and deterministic ordering; implement pagination/server filtering where supported.

Acceptance
- Unit tests: mapping utilities and validators; deterministic ordering verified.
- Docs: templates note persisted vs computed fields and model mapping.

Status
- Implemented for current items; continue applying to new resources/data sources.

---

## 4. Resources & Data Sources Roadmap (Section 18)

MVP (foundational coverage)
- Resources
  - jira_workflow_status — CRUD + import
  - jira_work_type — CRUD + import; standard and subtask
  - jira_project — CRUD + import
  - jira_priority — CRUD + import (custom priorities)
  - jira_resolution — CRUD + import (custom resolutions)
  - jira_project_component — CRUD + import (project-scoped)
  - jira_project_version — CRUD + import (project-scoped)
  - jira_issue_link_type — CRUD + import (global)
  - jira_workflow — CRUD + import; supports statuses, basic transitions, and a limited initial subset of validators/conditions; advanced/unsupported rules are read-only with warnings (no post-functions)
- Data sources
  - jira_work_types — list/lookup by ids or names; conflict validation
  - jira_workflow_statuses — list/lookup
  - jira_workflow — lookup by name or ID (base fields: name, description, statuses, basic transitions)
  - jira_project — lookup by key/ID
  - jira_projects — list/filter
  - jira_priorities — list
  - jira_resolutions — list
  - jira_issue_link_types — list
  - jira_fields — system and custom fields with IDs/types
  - jira_status_categories — list of status categories

Milestones
- Acceptance tests for all MVP items; tfplugindocs docs/examples validated; rate-limit handling validated on MVP endpoints.

v1.0 (schemes and admin breadth)
- Resources: custom fields + contexts/options; workflow scheme; issue type scheme; issue type screen scheme; screens + screen schemes; permission/notification/security schemes; project roles; role actors; webhooks; project category; workflow enhancements (incremental transitions support).
- Data sources: corresponding lookups/lists (custom_field(s), workflow(s), schemes, screens, permission/notification/security, roles, users/groups, field configurations, project components/versions, project features).

Future
- Association resources (bind scheme→project), filters and permissions, audit/time tracking settings, optional content resources gated by feature flags, Premium hierarchy resources.

Acceptance
- Roadmap items tracked with implementation, docs, and tests checkboxes; import formats documented per resource.

Status
- Current implementation covers a subset; expand per the above ordering guidance.

---

## 5. Data Source Enhancement: Hierarchy-aware Work Types (Section 23/24)

Actions
- Add attributes to jira_work_types:
  - include_subtasks (bool), top_level_only (bool)
  - allowed_levels (list(number))
  - hierarchy_min (number), hierarchy_max (number)
  - require_consistent_hierarchy (bool)
- Constraints/validation
  - include_subtasks and top_level_only are mutually exclusive with each other and with allowed_levels and hierarchy_min/max.
  - allowed_levels cannot be combined with hierarchy_min/max.
  - When both min and max are provided, enforce min ≤ max.
  - Use path-based diagnostics (AddAttributeError) with actionable messages.
- Behavior
  - Client-side filter by HierarchyLevel according to active predicate.
  - Warn when items are filtered out by hierarchy constraints for ids/names requests; continue with remaining.
  - If require_consistent_hierarchy is true and final selection mixes levels, return a diagnostic error.
  - Maintain deterministic ordering and stable ID keying.
- Tests & docs
  - Unit tests: constraints, filtering, warnings, consistency errors.
  - Acceptance tests: subtasks-only, top-level-only, explicit levels, min/max ranges (where feasible).
  - Update docs/templates with examples; regenerate tfplugindocs.

Acceptance
- Unit and acceptance tests added and passing; docs regenerated with no CI diff.

Status
- Planned workstream; not started.

---

## 6. Idempotence & Error Handling (FR-3, FR-6)

Actions
- CRUD uses canonical IDs; avoid diff-prone fields; normalize server values where safe.
- Centralize Atlassian error translation to actionable diagnostics; map 404 to disappearance, 400 to attribute errors.
- Implement and document import ID formats; validate during ImportState.

Acceptance
- Acceptance tests: CRUD + import and negative/permission scenarios across MVP.
- Unit tests: error mapping and import parsing.

Status
- Central helpers used; expand with new resources.

---

## 7. Rate Limits, Retries, and Concurrency (FR-6, NFR-2)

Actions
- Respect Retry-After and X-RateLimit headers; implement capped exponential backoff.
- Provider configuration exposes: retry_max_attempts, retry_initial_backoff, retry_max_backoff, http_timeout, respect_rate_limits (default true).
- Provide actionable guidance in diagnostics for 429s and README troubleshooting (tuning and parallelism tips).

Acceptance
- Unit tests simulate 429/5xx; CI artifacts include rate-limit incident counts.

Status
- Core policy implemented; add artifact reporting in CI.

---

## 8. Testing Strategy (FR-8, Section 13)

Actions
- Unit tests
  - Provider ValidateConfig/Configure: env precedence, defaults, validators, UA with version.
  - Wrapper helpers: status mapping, retries, conversions with fakes.
- Acceptance tests (TF_ACC=1)
  - CRUD + import for MVP resources; negative and permissions scenarios.
  - Data sources: ID/name filters, mixed-case handling, not-found behavior, hierarchy filters.
  - Sweeper to clean temporary resources; record flakiness metrics and 429 incidents in CI artifacts.
- Environment variables for TF_ACC
  - Required: JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
  - Optional (Basic): JIRA_USERNAME, JIRA_PASSWORD

Acceptance
- CI green on unit and docs; scheduled/protected-branch acceptance runs using CI secrets.

Status
- Existing tests in place for current items; expand to cover full MVP and new validations.

---

## 9. CI/CD, Release, and Supply Chain (Sections 12, 15, 17; NFR-6..NFR-12)

Actions
- CI guardrails
  - golangci-lint; unit tests with -race and coverage; Terraform version matrix for acceptance.
  - go generate ./... and fail on git diff (docs must be in sync).
  - Tidy check for go.mod/go.sum.
- Release notes automation and guardrails
  - Release Drafter with canonical labels and Keep a Changelog template.
  - PR label guard enforcing exactly one canonical category label or a skip label.
  - Changelog validator on tags with remediation guidance.
- Supply chain integrity and provenance
  - Signed commits/tags; GoReleaser artifacts with checksums and signatures.
  - SBOM generation and provenance attached to releases; multi-arch artifacts.
- Tooling and compatibility
  - Go >= 1.24; Terraform Core >= 1.6; pinned TPF, go-atlassian v2, tfplugindocs; Taskfile targets.
- Secrets/vuln management
  - Secret scanning in CI; govulncheck/OSV scans; Renovate/Dependabot automation.
- Governance and access controls
  - CODEOWNERS/branch protections; approvals for CI/release/security changes; avoid self-approval where feasible.

Acceptance
- CI workflows include above checks and fail on violations; release artifacts show checksums/signatures/SBOM/provenance links.

Status
- Lint/docs checks present; implement label guard, changelog validator, signing/SBOM/provenance, secret/vuln scans, and Renovate/Dependabot.

---

## 10. Documentation & Examples (FR-7, Section 14)

Actions
- tfplugindocs-generated docs for every public resource/data source with examples.
- README includes provider configuration, least-privilege token guidance, testing instructions, and troubleshooting:
  - 401/403 (auth/permissions), 404 (missing resources), 429 (rate limits with backoff tuning and parallelism), TLS/endpoint issues.
- Keep provider address consistent and examples usable.

Acceptance
- go generate ./... produces deterministic docs; no post-generate diff in CI.

Status
- Docs exist for current items; expand for new resources/data sources and hierarchy filters.

---

## 11. Observability & Logging (NFR-4)

Actions
- Structured debug logs (slog/JSON) gated by TF_LOG=DEBUG and optional provider debug toggle.
- Redact secrets globally; add request-scoped fields (resource IDs, operation, retries).

Acceptance
- Unit tests confirm redaction; manual smoke verifies no PII in debug logs.

Status
- Redaction in place; implement debug toggle and tests.

---

## 12. Performance & Pagination (NFR-3)

Actions
- Use server-side filters and pagination; avoid N+1; batch where possible.

Acceptance
- Data source tests confirm complete and ordered outputs.

Status
- Implemented for current lists; continue for new ones.

---

## 13. Compatibility & Version Pinning (NFR-5, Section 17)

Actions
- Pin dependency versions and document baselines; add upgrade tests when bumping.

Acceptance
- CI reproducibility and compatibility matrix aligned with go.mod/go.sum.

Status
- Pinned; automate updates via Renovate/Dependabot.

---

## 14. Metrics & Telemetry (Section 16)

Actions
- Track coverage, resource/data source coverage vs roadmap, flakiness, and rate-limit incidents in CI artifacts.

Acceptance
- CI artifacts show metrics; trends monitored over time.

Status
- Initial metrics present; expand per scope.

---

## 15. Security & Privacy (NFR-1, Section 10)

Actions
- Least-privilege token guidance; CI secrets for TF_ACC; no PII in logs/docs.
- Add schema validators for auth conflicts; sanitize free-form inputs where relevant.

Acceptance
- Unit tests for validators; docs updated; CI secrets masked.

Status
- In progress.

---

## 16. Risks & Mitigations (Section 19)

- Missing go-atlassian endpoints → Minimal direct HTTP in wrapper with tests; contribute upstream.
- API rate limits → Central backoff, scheduled acceptance tests, tuned parallelism; troubleshooting docs.
- Upstream breaks → Pin versions, isolate via wrapper, add upgrade tests.
- Validator gaps → Add validators and tests; actionable diagnostics.
- Redaction regressions → Unit-test redaction; review logs in acceptance runs.

---

## 17. Open Questions & Decision Log (Section 20)

Questions
- Baseline go-atlassian version and upgrade cadence.
- Provider-level concurrency tuning: expose or document Terraform -parallelism guidance only.
- Workflow initial subset and read-only behavior for advanced rules.

Decision Log Template
- Date, Context, Options, Decision, Rationale, Impact (schema/docs/tests).

---

## 18. Phased Milestones & Sequencing

Phase 0: Foundational plumbing — Completed
- Provider config refactor (env defaults, auth unification, timeouts, user-agent), wrapper, response handling, logging redaction, constants. Unit tests green; docs build.

Phase 1: Data stability and performance — Completed (current scope)
- Canonical IDs, normalization, deterministic ordering, pagination/filtering, sweeper, examples; rate-limit handling validated.

Phase 2: MVP resources and acceptance — In Progress
- Implement full MVP resource/data-source set listed in Section 4 with CRUD/import and acceptance tests.
- Plan and integrate governance/access controls, supply-chain integrity/provenance, secret/vuln management, auditability, privacy, policy mapping, and continuous controls validation.

Phase 3: CI/CD hardening and releases — In Progress
- Enforce lint/go generate no-diff, TF version matrix, race/coverage, GoReleaser; release automation (Release Drafter, label guard, changelog validator), signing, SBOM/provenance, dependency automation.

Phase 4: v1.0 breadth — Not Started
- Schemes/admin resources with robust diagnostics/imports; end-to-end templates.

---

## 19. Acceptance & Done Criteria (Sections 3, 22, 23)

- Build and test: unit + acceptance (where secrets available) pass across supported Terraform versions; -race enabled.
- Docs: tfplugindocs covers all public items; CI shows no post-generate diff.
- Client: go-atlassian primary integration; retry/backoff with rate-limit respect; UA embeds provider version.
- Tests: CRUD + import; negative/permissions paths; sweeper prevents leaks; metrics for flakiness/429s uploaded as CI artifacts.
- Security: sensitive values marked and redacted; env-driven config; README covers least-privilege and troubleshooting.
- Versioning: SemVer; binary embeds version; releases tagged with signed artifacts, checksums, SBOM, and provenance.

---

## 20. Immediate Next Actions (Next 1–2 weeks)

1) Release process docs + automation
   - Add Release Drafter config and workflow; implement PR label guard; add changelog validator on tags.
   - Acceptance: CI workflow files present, run on correct events, and pass in dry runs; CHANGELOG validated on tag.

2) Schema validators for auth conflicts and required-with semantics
   - Add validators for api_auth_email/api_token vs username/password; document env precedence in provider docs.
   - Acceptance: unit tests for conflict/required-with; docs updated.

3) Debug toggle with redaction tests
   - Implement provider-level debug toggle consistent with TF_LOG=DEBUG; ensure redaction across logs.
   - Acceptance: unit tests verify redaction; manual smoke logs show no secrets.

4) Dependency automation and tidy checks
   - Add Renovate/Dependabot; add go mod tidy check in CI; pin tool versions in go.mod as needed.
   - Acceptance: CI job fails on dirty go.mod/go.sum; Renovate/Dependabot PRs opened.

5) SBOM/provenance/signing configuration
   - Configure GoReleaser to produce signed, checksummed, multi-arch artifacts with SBOM and provenance; sign tags/commits.
   - Acceptance: release artifacts include signatures, checksums, SBOM, provenance links; verification steps documented in RELEASE.md.

6) Hierarchy-aware work types data source
   - Implement attributes, constraints, filtering, warnings, and consistency behavior.
   - Acceptance: unit + acceptance tests added; tfplugindocs regenerated without diff; examples updated.

Deliverables
- Mergeable PRs per workstream with tests and docs; CI green on lint/unit/docs; acceptance as secrets allow.

