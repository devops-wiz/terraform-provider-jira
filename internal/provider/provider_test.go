package provider

import (
	"context"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderSchema_ContainsCoreAttributes(t *testing.T) {
	p := &JiraProvider{version: "test"}
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)
	attrs := resp.Schema.Attributes
	if attrs == nil {
		t.Fatal("expected provider schema attributes to be set")
	}
	required := []string{
		"api_token", "api_auth_email", "username", "password",
		"http_timeout_seconds", "retry_max_attempts", "retry_initial_backoff_ms", "retry_max_backoff_ms",
		"email_redaction_mode",
	}
	for _, k := range required {
		if _, ok := attrs[k]; !ok {
			t.Fatalf("missing %s attribute in schema", k)
		}
	}
}

func TestProviderConfig_ApiTokenEnvDefaults_Valid(t *testing.T) {
	t.Setenv("JIRA_ENDPOINT", "https://example.atlassian.net")
	t.Setenv("JIRA_API_EMAIL", "user@example.com")
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_123456")

	data := JiraProviderModel{
		Endpoint:              types.StringNull(),
		AuthMethod:            types.StringNull(), // defaults to api_token
		OperationTimeouts:     nil,
		HTTPTimeoutSeconds:    types.Int64Null(),
		RetryOn4295xx:         types.BoolNull(),
		RetryMaxAttempts:      types.Int64Null(),
		RetryInitialBackoffMs: types.Int64Null(),
		RetryMaxBackoffMs:     types.Int64Null(),
		APIToken:              types.StringNull(),
		APIAuthEmail:          types.StringNull(),
		Username:              types.StringNull(),
		Password:              types.StringNull(),
	}

	rc := deriveResolvedConfig(data)
	if rc.authMethod != "api_token" {
		t.Fatalf("expected default auth_method 'api_token', got %q", rc.authMethod)
	}
	if rc.endpoint == "" || rc.email == "" || rc.apiToken == "" {
		t.Fatalf("expected endpoint/email/token resolved from env, got: endpoint=%q email=%q token=%q", rc.endpoint, rc.email, rc.apiToken)
	}
	errs := validateResolvedConfig(rc)
	if len(errs) > 0 {
		t.Fatalf("expected no validation errors, got: %+v", errs)
	}
}

func TestProviderConfig_BasicExplicit_Valid(t *testing.T) {
	// Ensure env does not inject API token path
	t.Setenv("JIRA_API_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	data := JiraProviderModel{
		Endpoint:              types.StringValue("https://example.local"),
		AuthMethod:            types.StringValue("basic"),
		OperationTimeouts:     nil,
		HTTPTimeoutSeconds:    types.Int64Value(30),
		RetryOn4295xx:         types.BoolValue(true),
		RetryMaxAttempts:      types.Int64Value(3),
		RetryInitialBackoffMs: types.Int64Value(200),
		RetryMaxBackoffMs:     types.Int64Value(2000),
		APIToken:              types.StringNull(),
		APIAuthEmail:          types.StringNull(),
		Username:              types.StringValue("example-user"),
		Password:              types.StringValue("EXAMPLE_PASSWORD_123!@#"),
	}

	rc := deriveResolvedConfig(data)
	if rc.authMethod != "basic" {
		t.Fatalf("expected auth_method 'basic', got %q", rc.authMethod)
	}
	errs := validateResolvedConfig(rc)
	if len(errs) > 0 {
		t.Fatalf("expected no validation errors, got: %+v", errs)
	}
}

func TestProviderConfig_MissingEndpoint_Err(t *testing.T) {
	// Ensure env is not providing endpoint
	t.Setenv("JIRA_ENDPOINT", "")

	data := JiraProviderModel{
		Endpoint:   types.StringNull(),
		AuthMethod: types.StringValue("basic"),
		Username:   types.StringValue("example-user"),
		Password:   types.StringValue("EXAMPLE_PASSWORD_123!@#"),
	}

	rc := deriveResolvedConfig(data)
	errs := validateResolvedConfig(rc)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors, got none")
	}
	found := false
	for _, e := range errs {
		if e.attr == attrEndpoint {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error on attribute %q, got: %+v", attrEndpoint, errs)
	}
}

func TestProviderConfig_ConflictingCredentials_Err(t *testing.T) {
	data := JiraProviderModel{
		Endpoint:     types.StringValue("https://example.atlassian.net"),
		AuthMethod:   types.StringNull(), // default api_token
		APIAuthEmail: types.StringValue("user@example.com"),
		APIToken:     types.StringValue("EXAMPLE_TOKEN_abcd1234"),
		Username:     types.StringValue("example-user"), // conflict with email
	}

	rc := deriveResolvedConfig(data)
	errs := validateResolvedConfig(rc)
	if len(errs) < 2 {
		t.Fatalf("expected at least two validation errors for conflicting creds, got: %+v", errs)
	}
	var hasEmailPath, hasUsernamePath bool
	for _, e := range errs {
		if e.attr == attrAPIAuthEmail {
			hasEmailPath = true
		}
		if e.attr == attrUsername {
			hasUsernamePath = true
		}
	}
	if !hasEmailPath || !hasUsernamePath {
		t.Fatalf("expected errors on both %q and %q, got: %+v", attrAPIAuthEmail, attrUsername, errs)
	}
}

func TestProviderConfig_InvalidTimeoutAndRetry_Err(t *testing.T) {
	data := JiraProviderModel{
		Endpoint:              types.StringValue("https://example.atlassian.net"),
		AuthMethod:            types.StringValue("api_token"),
		APIAuthEmail:          types.StringValue("user@example.com"),
		APIToken:              types.StringValue("EXAMPLE_TOKEN_abcd1234"),
		HTTPTimeoutSeconds:    types.Int64Value(0), // invalid
		RetryOn4295xx:         types.BoolValue(true),
		RetryMaxAttempts:      types.Int64Value(0),      // invalid
		RetryInitialBackoffMs: types.Int64Value(700000), // invalid (too large)
		RetryMaxBackoffMs:     types.Int64Value(100),
	}

	rc := deriveResolvedConfig(data)
	errs := validateResolvedConfig(rc)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors, got none")
	}
	var hasHTTPTimeout, hasRetryAttempts, hasRetryInitial bool
	for _, e := range errs {
		switch e.attr {
		case attrHTTPTimeoutSeconds:
			hasHTTPTimeout = true
		case attrRetryMaxAttempts:
			hasRetryAttempts = true
		case attrRetryInitialBackoff:
			hasRetryInitial = true
		}
	}
	if !(hasHTTPTimeout && hasRetryAttempts && hasRetryInitial) {
		t.Fatalf("expected errors on http timeout and retry settings, got: %+v", errs)
	}
}

func TestProviderConfig_RetryInitialGreaterThanMax_Err(t *testing.T) {
	data := JiraProviderModel{
		Endpoint:              types.StringValue("https://example.atlassian.net"),
		AuthMethod:            types.StringValue("api_token"),
		APIAuthEmail:          types.StringValue("user@example.com"),
		APIToken:              types.StringValue("EXAMPLE_TOKEN_abcd1234"),
		RetryOn4295xx:         types.BoolValue(true),
		RetryMaxAttempts:      types.Int64Value(2),
		RetryInitialBackoffMs: types.Int64Value(6000),
		RetryMaxBackoffMs:     types.Int64Value(5000),
	}

	rc := deriveResolvedConfig(data)
	errs := validateResolvedConfig(rc)
	if len(errs) == 0 {
		t.Fatalf("expected validation errors, got none")
	}
	found := false
	for _, e := range errs {
		if e.attr == attrRetryInitialBackoff {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected error on %q when initial backoff > max", attrRetryInitialBackoff)
	}
}

func TestProviderConfig_NoSecretLeakInMessages(t *testing.T) {
	secret := "EXAMPLE_SECRET_VALUE"
	data := JiraProviderModel{
		Endpoint:   types.StringValue("https://example.atlassian.net"),
		AuthMethod: types.StringValue("basic"),
		Username:   types.StringValue("example-user"),
		Password:   types.StringValue(secret),
	}

	rc := deriveResolvedConfig(data)
	errs := validateResolvedConfig(rc)
	for _, e := range errs {
		if e.detail != "" && (strings.Contains(e.detail, secret) || strings.Contains(e.summary, secret)) {
			t.Fatalf("validation error exposes secret in message: %+v", e)
		}
	}
}

func TestProviderConfig_EndpointAliasOnly_UsesAlias(t *testing.T) {
	// Canonical not set; alias provided
	t.Setenv("JIRA_ENDPOINT", "")
	t.Setenv("JIRA_BASE_URL", "https://alias.atlassian.net")
	// Auth via canonical envs
	t.Setenv("JIRA_API_EMAIL", "user@example.com")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_abcd1234")

	data := JiraProviderModel{
		Endpoint:   types.StringNull(),
		AuthMethod: types.StringNull(), // default api_token
	}
	rc := deriveResolvedConfig(data)
	if rc.endpoint != "https://alias.atlassian.net" {
		t.Fatalf("expected endpoint from alias, got %q", rc.endpoint)
	}
	if len(validateResolvedConfig(rc)) > 0 {
		t.Fatalf("expected no validation errors for alias endpoint config")
	}
}

func TestProviderConfig_EmailAliasOnly_UsesAlias(t *testing.T) {
	// Endpoint canonical
	t.Setenv("JIRA_ENDPOINT", "https://example.atlassian.net")
	// Canonical email not set; alias provided
	t.Setenv("JIRA_API_EMAIL", "")
	t.Setenv("JIRA_EMAIL", "alias@example.com")
	// Token canonical
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_abcd1234")

	data := JiraProviderModel{Endpoint: types.StringNull(), AuthMethod: types.StringNull()}
	rc := deriveResolvedConfig(data)
	if rc.email != "alias@example.com" {
		t.Fatalf("expected email from alias, got %q", rc.email)
	}
	if len(validateResolvedConfig(rc)) > 0 {
		t.Fatalf("expected no validation errors for alias email config")
	}
}

func TestProviderConfig_EnvPrecedence_CanonicalBeatsAlias(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "https://alias.atlassian.net")
	t.Setenv("JIRA_ENDPOINT", "https://canonical.atlassian.net")
	t.Setenv("JIRA_API_EMAIL", "user@example.com")
	t.Setenv("JIRA_EMAIL", "alias@example.com")
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_abcd1234")

	data := JiraProviderModel{Endpoint: types.StringNull(), AuthMethod: types.StringNull()}
	rc := deriveResolvedConfig(data)
	if rc.endpoint != "https://canonical.atlassian.net" {
		t.Fatalf("expected canonical endpoint to win, got %q", rc.endpoint)
	}
	if rc.email != "user@example.com" {
		t.Fatalf("expected canonical email to win, got %q", rc.email)
	}
}

func TestProviderConfig_HCLOverridesEnv(t *testing.T) {
	t.Setenv("JIRA_ENDPOINT", "https://canonical.atlassian.net")
	t.Setenv("JIRA_BASE_URL", "https://alias.atlassian.net")
	t.Setenv("JIRA_API_EMAIL", "user@example.com")
	t.Setenv("JIRA_EMAIL", "alias@example.com")
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_abcd1234")

	data := JiraProviderModel{Endpoint: types.StringValue("https://hcl.local"), AuthMethod: types.StringNull()}
	rc := deriveResolvedConfig(data)
	if rc.endpoint != "https://hcl.local" {
		t.Fatalf("expected HCL endpoint to override envs, got %q", rc.endpoint)
	}
}

func TestProviderConfig_Defaults_HTTP_Retry(t *testing.T) {
	// Minimal env to satisfy API token path
	// Endpoint and credentials via env, optional knobs unset in HCL
	t.Setenv("JIRA_ENDPOINT", "https://example.atlassian.net")
	t.Setenv("JIRA_API_EMAIL", "user@example.com")
	t.Setenv("JIRA_API_TOKEN", "EXAMPLE_TOKEN_123456")

	data := JiraProviderModel{
		Endpoint:              types.StringNull(),
		AuthMethod:            types.StringNull(), // should default to api_token
		HTTPTimeoutSeconds:    types.Int64Null(),
		RetryOn4295xx:         types.BoolNull(),
		RetryMaxAttempts:      types.Int64Null(),
		RetryInitialBackoffMs: types.Int64Null(),
		RetryMaxBackoffMs:     types.Int64Null(),
		APIToken:              types.StringNull(),
		APIAuthEmail:          types.StringNull(),
		Username:              types.StringNull(),
		Password:              types.StringNull(),
	}

	rc := deriveResolvedConfig(data)
	if rc.authMethod != "api_token" {
		t.Fatalf("expected default auth_method 'api_token', got %q", rc.authMethod)
	}
	if rc.httpTimeoutSeconds != 30 {
		t.Fatalf("expected default http_timeout_seconds 30, got %d", rc.httpTimeoutSeconds)
	}
	if rc.retryOn4295xx != true {
		t.Fatalf("expected default retry_on_429_5xx true, got %v", rc.retryOn4295xx)
	}
	if rc.retryMaxAttempts != 4 {
		t.Fatalf("expected default retry_max_attempts 4, got %d", rc.retryMaxAttempts)
	}
	if rc.retryInitialBackoffMs != 500 {
		t.Fatalf("expected default retry_initial_backoff_ms 500, got %d", rc.retryInitialBackoffMs)
	}
	if rc.retryMaxBackoffMs != 5000 {
		t.Fatalf("expected default retry_max_backoff_ms 5000, got %d", rc.retryMaxBackoffMs)
	}
	if rc.emailRedactionMode != "full" {
		t.Fatalf("expected default email_redaction_mode 'full', got %q", rc.emailRedactionMode)
	}
}

// New tests for email redaction mode precedence and behavior

func TestEmailRedactionMode_Prec_HCLOverridesEnv(t *testing.T) {
	t.Setenv("JIRA_EMAIL_REDACTION_MODE", "mask")
	data := JiraProviderModel{
		EmailRedactionMode: types.StringValue("full"),
	}
	rc := deriveResolvedConfig(data)
	if rc.emailRedactionMode != "full" {
		t.Fatalf("HCL should override env: expected 'full', got %q", rc.emailRedactionMode)
	}
}

func TestEmailRedactionMode_Prec_EnvWhenUnsetInHCL(t *testing.T) {
	t.Setenv("JIRA_EMAIL_REDACTION_MODE", "mask")
	data := JiraProviderModel{
		EmailRedactionMode: types.StringNull(),
	}
	rc := deriveResolvedConfig(data)
	if rc.emailRedactionMode != "mask" {
		t.Fatalf("Env should apply when HCL unset: expected 'mask', got %q", rc.emailRedactionMode)
	}
}

func TestSanitizeEmail_FullMode(t *testing.T) {
	out := sanitizeEmail("user@example.com", "full")
	if out != "[REDACTED_EMAIL]" {
		t.Fatalf("expected full redaction, got %q", out)
	}
	// Non-standard email should still redact fully
	out2 := sanitizeEmail("not-an-email", "full")
	if out2 != "[REDACTED_EMAIL]" {
		t.Fatalf("expected full redaction for invalid email, got %q", out2)
	}
}

func TestSanitizeEmail_MaskMode(t *testing.T) {
	out := sanitizeEmail("user@example.com", "mask")
	if out != "a****@example.com" {
		t.Fatalf("expected masked email 'a****@example.com', got %q", out)
	}
	// Invalid email falls back to full redaction
	out2 := sanitizeEmail("invalid", "mask")
	if out2 != "[REDACTED_EMAIL]" {
		t.Fatalf("expected '[REDACTED_EMAIL]' for invalid email in mask mode, got %q", out2)
	}
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.

	if v := os.Getenv("JIRA_ENDPOINT"); v == "" {
		t.Fatal("JIRA_ENDPOINT must be set for acceptance tests")
	} else {
		u, err := url.Parse(v)
		if err != nil {
			t.Fatalf("JIRA_ENDPOINT is not a valid URL: %v", err)
		}
		// Require https, non-empty host, and no embedded credentials, query, or fragment
		if u.Scheme != "https" {
			t.Fatal("JIRA_ENDPOINT must use https scheme")
		}
		if u.Host == "" {
			t.Fatal("JIRA_ENDPOINT must include a host (e.g., https://<tenant>.atlassian.net)")
		}
		if u.User != nil {
			t.Fatal("JIRA_ENDPOINT must not include credentials")
		}
		if u.RawQuery != "" || u.Fragment != "" {
			t.Fatal("JIRA_ENDPOINT must not include query parameters or fragments")
		}
		// Avoid local endpoints by mistake
		if strings.Contains(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.") {
			t.Fatal("JIRA_ENDPOINT must not point to localhost")
		}
	}

	if v := os.Getenv("JIRA_API_EMAIL"); v == "" {
		t.Fatal("JIRA_API_EMAIL must be set for acceptance tests")
	} else {
		// Basic email sanity check without echoing PII on failure
		if len(v) > 254 {
			t.Fatal("JIRA_API_EMAIL appears too long")
		}
		re := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
		if !re.MatchString(v) {
			t.Fatal("JIRA_API_EMAIL must be a valid email address")
		}
	}

	if v := os.Getenv("JIRA_API_TOKEN"); v == "" {
		t.Fatal("JIRA_API_TOKEN must be set for acceptance tests")
	} else {
		// Token should have a reasonable length, no whitespace, and not be a placeholder
		if len(v) < 8 {
			t.Fatal("JIRA_API_TOKEN appears too short")
		}
		if strings.ContainsAny(v, " \t\r\n") {
			t.Fatal("JIRA_API_TOKEN must not contain whitespace")
		}
		lower := strings.ToLower(v)
		if lower == "changeme" || strings.Contains(lower, "example") || strings.Contains(lower, "token") {
			t.Fatal("JIRA_API_TOKEN must not be a placeholder value")
		}
	}
}

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"jira": providerserver.NewProtocol6WithError(New("test")()),
}
