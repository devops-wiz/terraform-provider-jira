# Requirements: Terraform Provider for Atlassian Jira Cloud

Project: terraform-provider-jira (Terraform Plugin Framework + go-atlassian)
Author: DevOps Wiz Platform Team

## 1. Project overview
This repository implements a Terraform provider for Atlassian Jira Cloud using the Terraform Plugin Framework. It uses the go-atlassian client (https://github.com/ctreminiom/go-atlassian) to call Jira Cloud REST APIs and maps Jira models to provider resources and data sources for reproducible, auditable configuration-as-code. See Jira REST API v3 overview: https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/

Primary goals:
- Provide an idiomatic Terraform Plugin Framework provider for Jira Cloud using go-atlassian.
- Expose core Jira Cloud configuration (issue types/work types, workflow statuses, workflows, projects where safe) as Terraform resources and data sources.
- Ship reliable acceptance tests, generated documentation, and clear guidance for Jira Premium features.

Registry address:
- registry.terraform.io/devops-wiz/jira

## 2. High-level goals
- Accurate mapping of go-atlassian models to Terraform schema attributes (validated and documented).
- Safe and idempotent operations using go-atlassian service clients and canonical Jira IDs.
- Support API token (primary), optionally Basic auth, with a migration path to OAuth/App auth (not yet implemented).
- Generate tfplugindocs documentation and keep examples synchronized with code.

## 3. Success criteria / measurable outcomes
- Provider builds cleanly and passes unit + acceptance tests in CI (TF_ACC on).
- tfplugindocs output covers all public resources/data sources with examples.
- go-atlassian integration is used for API interactions (no ad-hoc HTTP except when strictly necessary).
- Acceptance tests validate server-side state using go-atlassian models.
- Users can manage selected Jira configuration end-to-end via Terraform without manual steps.

## 4. Stakeholders
Given team size and current ownership, most roles are fulfilled by the same person. To reflect that reality and reduce ceremony, we consolidate stakeholder roles as follows:

- Maintainer-Operator (primary, single owner)
  - Responsibilities: development, reviews, release management, CI configuration, security posture, and acceptance testing enablement.
  - Acts as: provider maintainer/contributor, QA (acceptance testing), security reviewer for credentials handling, and CI/CD operator.
- End users (consumers of the provider)
  - Platform/DevOps engineers and Jira admins who write Terraform to manage Jira configuration.
  - Provide feedback via issues/PRs; may run acceptance tests in their own Jira Cloud test sites.

Implications:
- Approvals: single-approver flow is acceptable until contributor base expands. When a second maintainer joins, switch to 2-person review for release PRs.
- Releases: the Maintainer-Operator owns tagging, changelog updates, and registry publishing.
- Security: the Maintainer-Operator ensures secrets are masked in CI and that docs avoid PII.
- Availability: document a fallback plan (e.g., a secondary contact or pause releases) during maintainer absence.

Maintainers:
- @devops-wiz/terraform-provider-jira maintainers (DevOps Wiz Platform Team)

## 5. Users and personas
- Platform engineer: automate and standardize Jira configuration across environments.
- Jira admin/team lead: enforce consistent issue types, fields, workflows, and permissions.
- Contributor: extend provider safely with clear patterns and examples.

## 6. Scope and out-of-scope
In-scope:
- Resources/data sources backed by go-atlassian for common Jira Cloud admin entities (work types, workflow statuses, fields, screens, workflows, schemes, project-scoped items).
- Provider configuration to initialize go-atlassian client via API token (primary) and optionally Basic auth.
- Acceptance tests interacting with a real Jira Cloud instance (using env-provided credentials).

Out-of-scope (initial):
- Implementing every Jira API surface immediately—prioritize high-value admin entities.
- Deep workflow authoring features (advanced conditions/validators/post-functions) at first release.
- UI tools or automation beyond provider responsibilities.

## 7. Functional requirements (go-atlassian specifics)
FR-1: Provider configuration & client construction
- Attributes:
  - endpoint (string) — Jira Cloud base URL (e.g., https://<your-org>.atlassian.net)
  - api_auth_email (sensitive string), api_token (sensitive string)
  - username (sensitive string), password (sensitive string) — only for basic auth if allowed
  - http_timeout (duration) — default "30s"
  - retry_max_attempts (number) — default 4
  - retry_initial_backoff (duration) — default "1s"
  - retry_max_backoff (duration) — default "30s"
  - respect_rate_limits (bool) — default true
- Environment variable mapping (canonical, with aliases):
  - Canonical: JIRA_ENDPOINT -> endpoint; JIRA_API_EMAIL -> api_auth_email; JIRA_API_TOKEN -> api_token
  - Aliases (lower precedence): JIRA_BASE_URL -> endpoint; JIRA_EMAIL -> api_auth_email
  - Precedence: canonical > alias (documented in schema; alias reading may be supported in configuration logic)
- Initialize go-atlassian jira.Client during Configure with correct auth and endpoint.
- Support env-based configuration for acceptance tests and CI.

FR-2: Encapsulated client wrapper
- Shared wrapper around go-atlassian to:
  - Centralize retry/backoff and rate-limit handling.
  - Convert go-atlassian models to Terraform state structures.
  - Provide helpers reused by resources (e.g., ID parsing, error wrapping).

FR-3: Resource lifecycle mapped to go-atlassian
- CRUD operations call appropriate go-atlassian service methods; parse responses into state.
- Import uses canonical Jira IDs; state holds stable identifiers.
- Validate inputs; fail fast with clear diagnostics.

FR-4: Data sources using go-atlassian
- List/lookup endpoints for existing configuration (work types, statuses, fields, etc.).
- Support filters for list endpoints if available from Jira API; validate mutually exclusive arguments; paginate where necessary.

FR-5: Idempotence & state accuracy
- Store canonical IDs and avoid ephemeral server fields in state.
- Normalize values (e.g., casing for categories) to prevent unnecessary diffs.

FR-6: Diagnostics & error mapping
- Translate Atlassian errors and HTTP status codes into actionable diagnostics.
- Handle 429s using headers (retry-after/backoff), and surface user guidance.

FR-7: Documentation & examples
- Use tfplugindocs to generate resource/data source docs with working examples.
- Provide example configs for common scenarios and Jira Premium notes.

FR-8: Tests
- Unit tests for model-to-state mapping and validators.
- Acceptance tests (TF_ACC) for CRUD + import; configurable via environment variables.

## 8. Non-functional requirements (NFRs)
NFR-1: Security
- Mark credentials as sensitive; never log secret values.
- Use CI secrets for acceptance tests; avoid storing credentials in files.

NFR-2: Reliability & rate limits
- Implement retry/backoff for transient errors; respect rate-limit headers.
- Ensure operations are idempotent and safe to retry.

NFR-3: Performance
- Minimize API calls; use pagination efficiently; avoid N+1 patterns.

NFR-4: Observability
- Provide debug logs (when enabled) with request identifiers and resource IDs.
- Enable debugging via TF_LOG=DEBUG. Secrets are redacted from logs.
- Structure logs for CI troubleshooting.

NFR-5: Compatibility
- Pin go-atlassian version; document supported API surfaces and upgrade steps.
- Match Terraform Plugin Framework best practices and version guidelines.

NFR-6: Governance and access controls
- Enforce CODEOWNERS and approvals for sensitive areas; branch protections on main/release branches.
- Require multiple approvals for CI/release/security changes; prevent self-approval where feasible.

NFR-7: Supply chain integrity and provenance
- Require signed commits/tags; publish checksums and signatures for artifacts.
- Generate SBOM per release and attach provenance; aim for reproducible, pinned builds and multi-arch artifacts.

NFR-8: Secrets, dependency, and vulnerability management
- Enable secret scanning pre-commit and in CI; prefer OIDC for CI with least privilege.
- Run vulnerability scans (e.g., govulncheck/OSV); automate dependency updates; enforce go.mod/go.sum hygiene.

NFR-9: Auditability and evidence
- Define retention policy for audit artifacts; automate periodic evidence exports.
- Enrich release notes with CI run IDs, checksums, SBOM, and provenance links.

NFR-10: Privacy and data handling
- Prohibit PHI/PII in repo; provide SECURITY.md/PRIVACY.md; ensure encryption for CI secrets/artifacts.
- Document data handling guidance for users and provide redaction/rotation examples.

NFR-11: Policy mapping and enterprise consumption
- Publish control mappings (SOX/SOC 2/HIPAA) and evidence links.
- Define LTS/support and deprecation policies; provide organization policy templates.

NFR-12: Continuous controls validation
- Schedule controls verification (branch protections, signing, scans, Actions hardening) and open issues on drift.

## 9. Constraints and assumptions
- Target: Jira Cloud REST API via go-atlassian (cloud semantics may differ from Server/DC).
- Some endpoints require admin privileges—tests must run with correctly scoped accounts.
- API rate limits are enforced by Atlassian; provider must behave politely and predictably.

## 10. Security & privacy considerations
- Document least-privilege token creation for users.
- Use environment variables in CI for TF_ACC tests:
  - JIRA_ENDPOINT (e.g., https://<your-org>.atlassian.net)
  - JIRA_API_EMAIL
  - JIRA_API_TOKEN
  - Optional for Basic: JIRA_USERNAME, JIRA_PASSWORD
  - Note: Aliases JIRA_BASE_URL (for endpoint) and JIRA_EMAIL (for api_auth_email) may be recognized at lower precedence; canonical vars take precedence.
- Avoid PII in logs and documentation outputs.

## 11. Data model & state mapping
- Use canonical Jira IDs (accountId, scheme/field IDs, projectId/key) in state.
- Document which nested model fields are persisted vs. computed/read-only.
- Consider storing provider version and endpoint info in diagnostics (not in state) for troubleshooting.

Import ID formats (general)
- jira_workflow_status — import by status ID
- jira_work_type — import by issue type ID
- Project-scoped resources (e.g., version, component) — import by ID (and project key where required by API); exact formats documented per resource page.

## 12. Integration points / external APIs
- go-atlassian (github.com/ctreminiom/go-atlassian/v2) as the Jira Cloud client.
- Atlassian Jira Cloud REST API (v3) endpoints via go-atlassian services (https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/).
- tfplugindocs for documentation generation; Task runner for build/test workflows.
- CI: GitHub Actions for lint, unit tests, doc validation, and (optionally) scheduled acceptance tests.

## 13. Acceptance tests & examples
- Acceptance tests toggle via TF_ACC=1 and use env vars for credentials/endpoint.
- Tests assert Terraform state and server state via go-atlassian models and responses.
- Provide examples:
  - jira_workflow_status: create/update/import, categories TODO/IN_PROGRESS/DONE.
  - jira_work_type: standard vs subtask with hierarchy_level; Premium guidance.
  - Data sources demonstrating ID and name lookups and conflict validation.

Troubleshooting (common cases)
- 401/403 Unauthorized/Forbidden: verify JIRA_API_EMAIL and JIRA_API_TOKEN; ensure required admin scopes.
- 404 Not Found: confirm project key/ID and resource existence in target site; check permissions.
- 429 Rate Limited: provider retries with backoff; consider reducing concurrency or scheduling runs. See Guidelines for details on retry behavior.
- TLS/Endpoint errors: verify JIRA_ENDPOINT and endpoint configuration.

### Rate limiting and retries (examples)

When TF_LOG=DEBUG is set, you’ll see structured messages indicating retry/backoff decisions. The provider respects Retry-After (when sent) and otherwise uses exponential backoff with jitter, up to retry_max_attempts. You can tune retry_initial_backoff, retry_max_backoff, and retry_max_attempts in provider configuration. For highly parallel plans, reduce Terraform parallelism (e.g., terraform apply -parallelism=5).

Sample: 429 with Retry-After header (success after backoff)
- Generated docs for every public resource/data source with:
  - Example Usage
  - Schema (Required/Optional/Read-Only)
  - Notes mapping attributes to Jira/go-atlassian models
  - Premium behavior notes where applicable
- README with provider configuration, auth methods, and testing instructions.
- Troubleshooting: common permission errors, rate limiting, and endpoint mismatches.

## 15. Developer tooling & automation
- go:generate for terraform fmt (examples) and tfplugindocs generation/validation.
- Task runner targets for build, lint, test (unit/acceptance), docs, and delve debugging.
- CI (GitHub Actions):
  - Lint (golangci-lint)
  - Unit tests
  - tfplugindocs validation
  - Optional acceptance tests on protected branches/schedules using CI secrets (JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN, JIRA_USERNAME, JIRA_PASSWORD)
- Release notes automation and guardrails:
  - Release Drafter configuration: `.github/release-drafter.yml` with canonical labels and Keep a Changelog template.
  - Workflow: `.github/workflows/release-drafter.yml` guarded to execute push events only on the repository default branch; PR events always run.
  - PR label guard: `.github/workflows/pr-label-guard.yml` enforces exactly one canonical category label or a skip label (`skip-release-notes`/`skip-changelog`). Canonical labels: `type:feat`, `type:change`, `type:refactor`, `type:docs`, `type:chore`, `type:test`, `type:deprecated`, `type:breaking`, `type:fix`, `security`.
  - CHANGELOG validator on tags: `.github/workflows/changelog-validate.yml` requires a Keep a Changelog–compliant version heading and provides remediation guidance.
- Toolchain alignment before releases: ensure Go matches go.mod (1.24.x) and Terraform CLI matches the CI matrix; `tfenv` is recommended to pin a supported Terraform version.
- Skipping heavy checks for non‑plugin changes: permitted when only docs/CI/chore areas are modified (see .junie/guidelines.md and CONTRIBUTING.md). Do not skip when `.go` files, modules, internal code, templates, or Taskfile behavior changes are involved.
- Examples rule: whenever `examples/**` changes, run `task gen` to reformat examples and re-generate/validate docs.

## 16. Metrics and telemetry (developer-facing)
- Track unit and acceptance test coverage.
- Track resource/data source coverage and gaps vs. Jira API.
- Monitor acceptance test flakiness and rate-limit incidents.

## 17. Dependencies & external services (compatibility matrix)
- Go: >= 1.24
- Terraform Core (tested): >= 1.6
- Terraform Plugin Framework: pinned (document the version from go.mod, e.g., v1.x.y)
- go-atlassian (github.com/ctreminiom/go-atlassian/v2): pinned major v2 (document the version from go.mod, e.g., v2.x.y)
- tfplugindocs (hashicorp/terraform-plugin-docs): pinned (document from go.mod)
- Atlassian Jira Cloud instance for acceptance tests

Pinning policy
- All library versions are pinned in go.mod. Upgrades are tested in CI and reflected in this section automatically (consider a CI job to update docs).

## 18. Roadmap checklist (Jira REST API v3 aligned)
Use this as a living checklist to track provider coverage. Items are grouped by phase and type (resources vs. data sources). Check off when implemented, documented (tfplugindocs), and covered by tests.

OpenAPI-aligned coverage note:
- The Jira Cloud REST v3 surface also exposes additional admin/configuration endpoints frequently used by Jira admins that are not yet listed below. Notable gaps include Field Configurations and their Schemes, Project Features toggles, Project and Issue Properties (key-value APIs), and Issue Security Level membership management. These have been added to the backlog sections to better reflect the REST v3 coverage.

Dependencies and ordering guidance (to avoid planning dead-ends)
- Project Features: Managing project components/versions may require enabling the corresponding project features. Ensure jira_project_feature is available before or alongside jira_project_component and jira_project_version (at least for enabling/disabling features).
- Roles before grants: Create and populate project roles (jira_project_role, jira_project_role_actors) before applying permission/notification/security scheme grants that reference those roles.
- Identity lookups: Data sources for users and groups (jira_users, jira_groups) should be available before resources that reference identities (permission schemes, notification schemes, issue security levels). Managing groups (jira_group, jira_group_members) is optional if groups are pre-existing.
- Field configurations: Create field configurations and schemes before attaching them to projects via the association resource.
- Security levels: Create an issue security scheme before managing its level members (jira_issue_security_level_members).

### MVP (foundational coverage)

Resources
- [ ] jira_workflow_status — CRUD + import; maps to workflow status entity
- [ ] jira_work_type (Issue Type) — CRUD + import; supports standard and subtask; Premium notes for hierarchy
- [ ] jira_project — CRUD + import; manage projects
- [ ] jira_priority — CRUD + import; manage custom priorities
- [ ] jira_resolution — CRUD + import; manage custom resolutions
- [ ] jira_project_component — CRUD + import; project-scoped
- [ ] jira_project_version — CRUD + import; project-scoped
- [ ] jira_issue_link_type — CRUD + import; global
- [ ] jira_workflow — CRUD + import; supports statuses, basic transitions, and a limited initial subset of validators/conditions; advanced/unsupported rules are read-only with warnings (no post-functions)

Data sources
- [ ] jira_work_types — list/lookup by ids or names; conflict validation
- [ ] jira_workflow_statuses — list/lookup statuses
- [ ] jira_workflow — lookup a workflow by name or ID (base fields: name, description, statuses, and basic transitions)
- [ ] jira_project — lookup by key/ID
- [ ] jira_projects — list/filter
- [ ] jira_priorities — read-only list
- [ ] jira_resolutions — read-only list
- [ ] jira_issue_link_types — read-only list
- [ ] jira_fields — list of system and custom fields with IDs/types
- [ ] jira_status_categories — read-only list of status categories (to aid status modeling)

Milestones
- [ ] Acceptance tests for all MVP items
- [ ] Docs and examples validated (tfplugindocs)
- [ ] Rate-limit handling validated against MVP endpoints

### v1.0 (schemes and admin breadth)

Resources
- [ ] jira_custom_field — selected field types (text, number, single/multi select, user, group)
- [ ] jira_custom_field_context — scopes and applicable issue types/projects
- [ ] jira_custom_field_option — enumeration management for select fields
- [ ] jira_workflow — enhance: add more transition features incrementally (still excluding advanced validators/conditions/post-functions at first)
- [ ] jira_workflow_scheme — mappings of issue types to workflows
- [ ] jira_issue_type_scheme — define allowed issue types per project
- [ ] jira_issue_type_scheme_project_association — attach issue type scheme to project
- [ ] jira_issue_type_screen_scheme — wire screens to issue types
- [ ] jira_screen — define screens (create/edit/view)
- [ ] jira_screen_scheme — map operations to screens
- [ ] jira_permission_scheme — permissions with grants
- [ ] jira_notification_scheme — notifications per events/recipients
- [ ] jira_issue_security_scheme — levels and memberships
- [ ] jira_project_role — create custom roles
- [ ] jira_project_role_actors — assign users/groups to roles
- [ ] jira_webhook — outbound webhooks
- [ ] jira_project_category — global categorization

Data sources
- [ ] jira_custom_field
- [ ] jira_custom_fields
- [ ] jira_workflow
- [ ] jira_workflows
- [ ] jira_workflow_scheme
- [ ] jira_workflow_schemes
- [ ] jira_issue_type_scheme
- [ ] jira_issue_type_schemes
- [ ] jira_issue_type_screen_scheme
- [ ] jira_screen
- [ ] jira_screens
- [ ] jira_screen_scheme
- [ ] jira_screen_schemes
- [ ] jira_permission_scheme
- [ ] jira_notification_scheme
- [ ] jira_issue_security_scheme
- [ ] jira_project_roles
- [ ] jira_groups
- [ ] jira_users
- [ ] jira_field_configurations
- [ ] jira_field_configuration_schemes
- [ ] jira_project_components — read-only list for a project (mirrors resource)
- [ ] jira_project_versions — read-only list for a project (mirrors resource)
- [ ] jira_project_features — list enabled/disabled features for a project

Milestones
- [ ] End-to-end “project template” flows (issue types + fields + screens + workflows + schemes)
- [ ] Import coverage for major schemes
- [ ] Robust diagnostics for permission errors and validation failures

### Future (expanded coverage and associations)

Resources
- [ ] jira_project_permissions_binding — attach permission scheme to project
- [ ] jira_workflow_scheme_project_association — bind workflow scheme to project
- [ ] jira_notification_scheme_project_association — bind notification scheme to project
- [ ] jira_issue_security_scheme_project_association — bind security scheme to project
- [ ] jira_filter — saved filter management
- [ ] jira_filter_permission — share/visibility for filters
- [ ] jira_issue_link — optional content-level linking (likely disabled by default)
- [ ] jira_audit_settings — read/update global audit settings
- [ ] jira_time_tracking_settings — global time tracking configuration
- [ ] Premium-only: jira_issue_type_hierarchy_level — custom hierarchy and mappings

Data sources
- [ ] jira_project_categories
- [ ] jira_project_role_details
- [ ] jira_audit_records
- [ ] jira_time_tracking_settings
- [ ] Premium-only: jira_issue_type_hierarchy
- [ ] jira_project_properties — list/read project properties
- [ ] jira_issue_properties — list/read issue properties

Milestones
- [ ] Association resources separated from scheme creation to minimize blast radius
- [ ] Optional “content” resources gated by feature flag/provider config
- [ ] Performance tuning and pagination for list-heavy data sources

Notes
- Prefer canonical IDs in state (projectId/key where appropriate, accountId for users, field IDs, scheme IDs).
- For users/groups, prefer accountId and group name; avoid email as an identifier.
- For workflows, start with a documented subset (statuses and basic transitions) and expand iteratively (validators, conditions, post-functions).
- Model associations as dedicated resources to improve composability and safety in CI/CD.

## 19. Risks & mitigation (go-atlassian specific)
- Missing endpoints in go-atlassian:
  - Mitigation: minimal direct HTTP in wrapper with tests; upstream contributions to go-atlassian.
- API rate limits causing flakiness:
  - Mitigation: central backoff, scheduled acceptance tests, and tuned concurrency.
- Breaking changes in go-atlassian:
  - Mitigation: pin versions, isolate usage via wrapper, add upgrade tests.

## 20. Implementation notes
- Shared client wrapper encapsulates client construction, retries, and model/state mapping.
- Provider configuration validates auth_method and marks credentials as sensitive.
- Use plan modifiers/validators to keep state minimal and meaningful.
- Keep doc templates aligned with model fields and Premium behaviors.
- Follow acceptance test patterns with import, update, and error-path coverage.

## 21. Acceptance checklist before v1.0 release
- [ ] go-atlassian client initialization implemented and tested (API token flow)
- [ ] Core resources (work types, workflow statuses) implemented with CRUD + import
- [ ] Unit tests for mapping and validators
- [ ] Acceptance tests using env credentials (TF_ACC) with reliable cleanup
- [ ] tfplugindocs output generated and validated
- [ ] README covers auth setup, provider config, examples, and troubleshooting

## 22. Versioning policy
- Semantic Versioning (SemVer) is used. Minor versions may add new resources/attributes; breaking changes only in major versions.
- Provider binary embeds version information for telemetry and support.


## 23. Future improvements (proposed)

- Work types data source: hierarchy-aware filtering and validation
  - Problem: Current client-side filtering ignores the issue type hierarchy (HierarchyLevel/Subtask). Users may unintentionally mix subtasks and top-level types in results, leading to grouping/consumption failures.
  - Proposal: Add optional attributes to control and validate hierarchy in the jira_work_types data source:
    - include_subtasks (bool): select only subtasks (level = -1).
    - top_level_only (bool): exclude subtasks (level != -1); document exact semantics.
    - allowed_levels (list(number)): explicit set of allowed levels (e.g., [-1, 0, 1]).
    - hierarchy_min (number), hierarchy_max (number): inclusive bounds for allowed levels.
    - require_consistent_hierarchy (bool): error if selected set contains multiple distinct hierarchy levels.
  - Constraints/validation:
    - include_subtasks/top_level_only are mutually exclusive with allowed_levels and with hierarchy_min/hierarchy_max.
    - allowed_levels should not be combined with hierarchy_min/hierarchy_max.
    - When both min and max are provided, require min ≤ max.
    - Use path-based diagnostics (AddAttributeError) for conflicts; keep messages actionable.
  - Behavior:
    - Apply client-side filtering over the fetched list by HierarchyLevel according to the active predicate.
    - For ids/names requests, warn when items are filtered out by hierarchy constraints; proceed with remaining.
    - If require_consistent_hierarchy = true and mixed levels are present in the final selection, return a diagnostic error.
    - Preserve current behavior by default (no constraints active).
  - Determinism & state:
    - Continue keying the map by stable IDs and maintain sorted/deterministic outputs.
  - Tests & docs:
    - Unit tests for constraints, filtering, warnings, and consistency checks.
    - Acceptance tests covering subtasks-only, top-level-only, explicit levels, and min/max ranges (where feasible).
    - Update templates/docs to describe new attributes and examples; regenerate tfplugindocs.
  - Security & performance:
    - No sensitive data introduced; logging must remain redacted.
    - Filtering is client-side on a single unpaginated list; no expected performance impact.
