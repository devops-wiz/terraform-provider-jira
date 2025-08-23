// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

// Package provider implements the Terraform Provider for Jira.
//
// Highlights:
//   - Auth: API token for Jira Cloud (recommended) and basic auth for self-hosted Jira.
//   - Timeouts & retries: configurable HTTP timeout and capped exponential backoff; honors Retry-After.
//   - Concurrency: safe for parallel plans; declare explicit dependencies when API ordering matters.
//   - Deterministic outputs: data sources/resources avoid spurious diffs via stable IDs and sorting.
//
// Further reading (canonical docs):
//   - Configuration & env vars: docs/index.md#configuration
//   - Retries, timeouts, and rate limits: docs/index.md#provider-retries-and-timeouts
//   - Troubleshooting: docs/index.md#troubleshooting
//   - Examples: docs/ and examples/
package provider
