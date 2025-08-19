// Package testhelpers provides shared testing utilities used across unit and
// acceptance tests.
//
// Intended use:
//   - Unit tests: builders, fixtures, and assertions to reduce boilerplate.
//   - Acceptance tests: environment pre-checks, deterministic name generators,
//     retry/backoff test stubs, and helpers for external service setup/teardown.
//   - HTTP/logging tests: redaction utilities, fake clients/round-trippers,
//     and golden-file helpers with stable, sorted outputs.
//
// Conventions:
//   - Keep dependencies minimal and avoid importing production-only paths.
//   - Ensure deterministic outputs: sort collections, use stable IDs, and
//     normalize values before comparisons or serialization.
//   - Never leak secrets in logs, errors, or golden files; always redact.
//
// This package is for test code and is not part of the provider's public API.
package testhelpers
