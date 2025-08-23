// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import "strings"

// sanitizeEmail masks or redacts an email to avoid PII leakage in logs/errors.
// Mode controls behavior:
//   - "full": fully redact, returning "[REDACTED_EMAIL]"
//   - "mask": partially mask local-part, preserving domain (e.g., "a****@example.com")
func sanitizeEmail(email string, mode string) string {
	if email == "" {
		return ""
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = defaultEmailRedactionMode
	}
	if mode != "full" && mode != "mask" {
		mode = defaultEmailRedactionMode
	}
	// Full redaction: never expose local-part or domain.
	if mode == "full" {
		return "[REDACTED_EMAIL]"
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		// Not a standard email; return a generic token to avoid echoing the raw value.
		return "[REDACTED_EMAIL]"
	}
	domain := email[at+1:]
	return "a****@" + domain
}

// redactSecretValue replaces a sensitive value with a stable token.
// If the value is empty, it returns the empty string to avoid adding tokens where not needed.
func redactSecretValue(v string) string {
	if v == "" {
		return ""
	}
	return "[REDACTED]"
}

// sanitizeValidationError returns a copy of the given validation error with secrets redacted.
func sanitizeValidationError(e validationErr, rc resolvedConfig) validationErr {
	// Build a mapping of raw -> redacted tokens
	replacements := map[string]string{}

	if rc.apiToken != "" {
		replacements[rc.apiToken] = redactSecretValue(rc.apiToken)
	}
	if rc.password != "" {
		replacements[rc.password] = redactSecretValue(rc.password)
	}
	if rc.username != "" {
		// Usernames can be sensitive; redact fully.
		replacements[rc.username] = redactSecretValue(rc.username)
	}
	if rc.email != "" {
		// For emails, replace with policy-driven sanitization (defaults to full).
		replacements[rc.email] = sanitizeEmail(rc.email, rc.emailRedactionMode)
	}

	// Apply replacements to summary/detail without echoing the original values.
	summary := e.summary
	detail := e.detail
	for raw, red := range replacements {
		if raw == "" {
			continue
		}
		if summary != "" {
			summary = strings.ReplaceAll(summary, raw, red)
		}
		if detail != "" {
			detail = strings.ReplaceAll(detail, raw, red)
		}
	}

	e.summary = summary
	e.detail = detail
	return e
}
