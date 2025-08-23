// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Test_deriveResolvedConfig_env_precedence_and_defaults(t *testing.T) {
	// Endpoint: prefer HCL over env; canonical > alias when HCL not set
	t.Run("endpoint canonical over alias; HCL overrides env", func(t *testing.T) {
		// alias set
		t.Setenv("JIRA_BASE_URL", "https://alias.atlassian.net")
		// canonical set
		t.Setenv("JIRA_ENDPOINT", "https://canon.atlassian.net")

		// HCL not set -> should pick canonical
		m := JiraProviderModel{Endpoint: types.StringNull()}
		rc := deriveResolvedConfig(m)
		if rc.endpoint != "https://canon.atlassian.net" {
			t.Fatalf("expected canonical endpoint, got %q", rc.endpoint)
		}

		// HCL set -> should override env
		m = JiraProviderModel{Endpoint: types.StringValue("https://hcl.atlassian.net")}
		rc = deriveResolvedConfig(m)
		if rc.endpoint != "https://hcl.atlassian.net" {
			t.Fatalf("expected HCL endpoint, got %q", rc.endpoint)
		}
	})

	t.Run("api email canonical over alias; defaults applied", func(t *testing.T) {
		t.Setenv("JIRA_EMAIL", "alias@example.com")
		t.Setenv("JIRA_API_EMAIL", "canon@example.com")
		t.Setenv("JIRA_API_TOKEN", "tok123")

		m := JiraProviderModel{APIAuthEmail: types.StringNull(), APIToken: types.StringNull(), AuthMethod: types.StringNull(), HTTPTimeoutSeconds: types.Int64Null(), RetryOn4295xx: types.BoolNull(), RetryMaxAttempts: types.Int64Null(), RetryInitialBackoffMs: types.Int64Null(), RetryMaxBackoffMs: types.Int64Null(), EmailRedactionMode: types.StringNull()}
		rc := deriveResolvedConfig(m)

		if rc.email != "canon@example.com" {
			t.Fatalf("expected canonical api email, got %q", rc.email)
		}
		if rc.apiToken != "tok123" {
			t.Fatalf("expected api token from env, got %q", rc.apiToken)
		}
		if rc.authMethod != defaultAuthMethod {
			t.Fatalf("expected default auth method %q, got %q", defaultAuthMethod, rc.authMethod)
		}
		if rc.httpTimeoutSeconds != defaultHTTPTimeoutSeconds {
			t.Fatalf("expected default http timeout %d, got %d", defaultHTTPTimeoutSeconds, rc.httpTimeoutSeconds)
		}
		if rc.retryOn4295xx != defaultRetryOn4295xx {
			t.Fatalf("expected default retryOn4295xx %v, got %v", defaultRetryOn4295xx, rc.retryOn4295xx)
		}
		if rc.retryMaxAttempts != defaultRetryMaxAttempts || rc.retryInitialBackoffMs != defaultRetryInitialBackoffMs || rc.retryMaxBackoffMs != defaultRetryMaxBackoffMs {
			t.Fatalf("expected retry defaults (%d,%d,%d), got (%d,%d,%d)", defaultRetryMaxAttempts, defaultRetryInitialBackoffMs, defaultRetryMaxBackoffMs, rc.retryMaxAttempts, rc.retryInitialBackoffMs, rc.retryMaxBackoffMs)
		}
		if rc.emailRedactionMode != defaultEmailRedactionMode {
			t.Fatalf("expected default email redaction %q, got %q", defaultEmailRedactionMode, rc.emailRedactionMode)
		}
	})

	t.Run("email redaction mode normalization", func(t *testing.T) {
		m := JiraProviderModel{EmailRedactionMode: types.StringValue(" MASK ")}
		rc := deriveResolvedConfig(m)
		if rc.emailRedactionMode != "mask" {
			t.Fatalf("expected mask, got %q", rc.emailRedactionMode)
		}
		m = JiraProviderModel{EmailRedactionMode: types.StringValue("invalid")}
		rc = deriveResolvedConfig(m)
		if rc.emailRedactionMode != defaultEmailRedactionMode {
			t.Fatalf("expected default %q, got %q", defaultEmailRedactionMode, rc.emailRedactionMode)
		}
	})
}

func Test_validateBase(t *testing.T) {
	t.Run("missing endpoint", func(t *testing.T) {
		rc := resolvedConfig{endpoint: "", authMethod: "api_token"}
		errs := validateBase(rc)
		if len(errs) == 0 {
			t.Fatalf("expected error for missing endpoint")
		}
		found := false
		for _, e := range errs {
			if e.attr == attrEndpoint {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected attr %s in errors", attrEndpoint)
		}
	})

	t.Run("invalid auth method", func(t *testing.T) {
		rc := resolvedConfig{endpoint: "https://x", authMethod: "oauth"}
		errs := validateBase(rc)
		found := false
		for _, e := range errs {
			if e.attr == attrAuthMethod {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected attr %s in errors", attrAuthMethod)
		}
	})
}

func Test_validateHTTP(t *testing.T) {
	for _, tt := range []struct {
		in      int
		wantErr bool
	}{
		{0, true}, {1, false}, {600, false}, {601, true},
	} {
		rc := resolvedConfig{httpTimeoutSeconds: tt.in}
		errs := validateHTTP(rc)
		if tt.wantErr && len(errs) == 0 {
			t.Fatalf("expected error for %d", tt.in)
		}
		if !tt.wantErr && len(errs) != 0 {
			t.Fatalf("expected no error for %d", tt.in)
		}
	}
}

func Test_validateRetry(t *testing.T) {
	t.Run("disabled returns no errors", func(t *testing.T) {
		rc := resolvedConfig{retryOn4295xx: false}
		errs := validateRetry(rc)
		if len(errs) != 0 {
			t.Fatalf("expected no errors when retries disabled, got %v", errs)
		}
	})

	t.Run("bounds and ordering", func(t *testing.T) {
		rc := resolvedConfig{retryOn4295xx: true, retryMaxAttempts: 0, retryInitialBackoffMs: 50, retryMaxBackoffMs: 40}
		errs := validateRetry(rc)
		if len(errs) < 3 {
			t.Fatalf("expected multiple errors, got %v", errs)
		}
		// fix to valid bounds and ensure no errors
		rc = resolvedConfig{retryOn4295xx: true, retryMaxAttempts: 3, retryInitialBackoffMs: 200, retryMaxBackoffMs: 1000}
		errs = validateRetry(rc)
		if len(errs) != 0 {
			t.Fatalf("expected no errors with valid retry settings, got %v", errs)
		}
	})
}

func Test_validateAuth(t *testing.T) {
	t.Run("conflicting creds", func(t *testing.T) {
		rc := resolvedConfig{email: "e@example.com", username: "u"}
		errs := validateAuth(rc)
		if len(errs) != 2 {
			t.Fatalf("expected 2 conflict errors, got %d", len(errs))
		}
	})

	t.Run("missing creds", func(t *testing.T) {
		rc := resolvedConfig{}
		errs := validateAuth(rc)
		if len(errs) != 2 {
			t.Fatalf("expected 2 missing errors, got %d", len(errs))
		}
	})

	t.Run("api_token path requirements and conflicts", func(t *testing.T) {
		rc := resolvedConfig{authMethod: "api_token", email: "", apiToken: "", username: "u", password: "p"}
		errs := validateAuth(rc)
		if len(errs) < 4 {
			t.Fatalf("expected >=4 errors for api_token path, got %d: %v", len(errs), errs)
		}
	})

	t.Run("basic path requirements and conflicts", func(t *testing.T) {
		rc := resolvedConfig{authMethod: "basic", username: "", password: "", email: "e@example.com", apiToken: "tok"}
		errs := validateAuth(rc)
		// expect missing username and password + two conflicts
		if len(errs) < 4 {
			t.Fatalf("expected >=4 errors for basic path, got %d: %v", len(errs), errs)
		}
	})
}

func Test_validateResolvedConfig_integration_and_redaction(t *testing.T) {
	// Construct an invalid config to trigger errors and ensure no secrets appear
	rc := resolvedConfig{
		endpoint:           "", // force base error to stop further checks
		authMethod:         "api_token",
		email:              "user@example.com",
		apiToken:           "super-secret-token",
		username:           "myuser",
		password:           "mypass",
		httpTimeoutSeconds: 30,
	}
	errs := validateResolvedConfig(rc)
	if len(errs) == 0 {
		t.Fatalf("expected at least one validation error")
	}
	// Although current messages do not include secrets, verify sanitizer does not leak values
	for _, e := range errs {
		if e.summary == "" && e.detail == "" {
			continue
		}
		if contains(e.summary, "super-secret-token") || contains(e.detail, "super-secret-token") || contains(e.summary, "myuser") || contains(e.detail, "myuser") || contains(e.summary, "mypass") || contains(e.detail, "mypass") || contains(e.summary, "user@example.com") || contains(e.detail, "user@example.com") {
			t.Fatalf("validation error leaked secret or email: %+v", e)
		}
	}
}

func contains(s, sub string) bool { return len(s) > 0 && len(sub) > 0 && (stringContains(s, sub)) }

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (time.Now().IsZero() == false) && (func() bool { return (len(s) > 0) && (len(sub) > 0) && (Index(s, sub) >= 0) })()
}

// Index is a tiny wrapper around strings.Index to avoid importing strings in this test file.
func Index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
