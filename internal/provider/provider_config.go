// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
)

// configuration derivation (unified) to avoid duplicated parsing across sections
func deriveResolvedConfig(data JiraProviderModel) resolvedConfig {
	// Base
	authMethod := readString(data.AuthMethod, "")
	if authMethod == "" {
		authMethod = defaultAuthMethod
	}
	endpoint := readStringWithAliases(data.Endpoint, "JIRA_ENDPOINT", "JIRA_BASE_URL")

	// Auth
	email := readStringWithAliases(data.APIAuthEmail, "JIRA_API_EMAIL", "JIRA_EMAIL")
	apiToken := readString(data.APIToken, "JIRA_API_TOKEN")
	username := readString(data.Username, "JIRA_USERNAME")
	password := readString(data.Password, "JIRA_PASSWORD")

	// HTTP
	httpTimeoutSeconds := readInt64Default(data.HTTPTimeoutSeconds, defaultHTTPTimeoutSeconds)

	// Retry
	retryOn4295xx := readBoolDefault(data.RetryOn4295xx, defaultRetryOn4295xx)
	retryMaxAttempts := readInt64Default(data.RetryMaxAttempts, defaultRetryMaxAttempts)
	retryInitialBackoffMs := readInt64Default(data.RetryInitialBackoffMs, defaultRetryInitialBackoffMs)
	retryMaxBackoffMs := readInt64Default(data.RetryMaxBackoffMs, defaultRetryMaxBackoffMs)

	// Privacy & Redaction
	mode := readString(data.EmailRedactionMode, "JIRA_EMAIL_REDACTION_MODE")
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" || (mode != "full" && mode != "mask") {
		mode = defaultEmailRedactionMode
	}

	return resolvedConfig{
		endpoint:              endpoint,
		authMethod:            authMethod,
		email:                 email,
		apiToken:              apiToken,
		username:              username,
		password:              password,
		httpTimeoutSeconds:    httpTimeoutSeconds,
		retryOn4295xx:         retryOn4295xx,
		retryMaxAttempts:      retryMaxAttempts,
		retryInitialBackoffMs: retryInitialBackoffMs,
		retryMaxBackoffMs:     retryMaxBackoffMs,
		emailRedactionMode:    mode,
	}
}

// validation per-section
func validateBase(rc resolvedConfig) []validationErr {
	var errs []validationErr
	if rc.endpoint == "" {
		errs = append(errs, validationErr{attr: attrEndpoint, summary: "Missing Endpoint Configuration.", detail: "Provide 'endpoint' or set JIRA_ENDPOINT (or JIRA_BASE_URL alias) environment variable."})
	}
	if rc.authMethod != "api_token" && rc.authMethod != "basic" {
		errs = append(errs, validationErr{attr: attrAuthMethod, summary: "Invalid Auth Method Configuration.", detail: "auth_method must be 'api_token' or 'basic'."})
	}
	return errs
}

func validateHTTP(rc resolvedConfig) []validationErr {
	if rc.httpTimeoutSeconds < 1 || rc.httpTimeoutSeconds > 600 {
		return []validationErr{{attr: attrHTTPTimeoutSeconds, summary: "Invalid HTTP Timeout Configuration.", detail: fmt.Sprintf("http_timeout_seconds must be between 1 and 600 seconds; got %d", rc.httpTimeoutSeconds)}}
	}
	return nil
}

func validateRetry(rc resolvedConfig) []validationErr {
	if !rc.retryOn4295xx {
		return nil
	}
	var errs []validationErr
	if rc.retryMaxAttempts < 1 || rc.retryMaxAttempts > 10 {
		errs = append(errs, validationErr{attr: attrRetryMaxAttempts, summary: "Invalid Retry Attempts Configuration.", detail: fmt.Sprintf("retry_max_attempts must be between 1 and 10; got %d", rc.retryMaxAttempts)})
	}
	if rc.retryInitialBackoffMs < 100 || rc.retryInitialBackoffMs > 600000 {
		errs = append(errs, validationErr{attr: attrRetryInitialBackoff, summary: "Invalid Retry Backoff Configuration.", detail: fmt.Sprintf("retry_initial_backoff_ms must be between 100 and 600000 milliseconds; got %d", rc.retryInitialBackoffMs)})
	}
	if rc.retryMaxBackoffMs < 100 || rc.retryMaxBackoffMs > 600000 {
		errs = append(errs, validationErr{attr: attrRetryMaxBackoff, summary: "Invalid Retry Backoff Configuration.", detail: fmt.Sprintf("retry_max_backoff_ms must be between 100 and 600000 milliseconds; got %d", rc.retryMaxBackoffMs)})
	}
	if rc.retryInitialBackoffMs > rc.retryMaxBackoffMs {
		errs = append(errs, validationErr{attr: attrRetryInitialBackoff, summary: "Invalid Retry Backoff Configuration.", detail: "retry_initial_backoff_ms must be less than or equal to retry_max_backoff_ms."})
	}
	return errs
}

func validateAuth(rc resolvedConfig) []validationErr {
	var errs []validationErr
	if rc.email != "" && rc.username != "" {
		return []validationErr{
			{attr: attrAPIAuthEmail, summary: "Conflicting credentials.", detail: "api_auth_email conflicts with username. Choose API token (api_auth_email + api_token) or basic (username + password), not both."},
			{attr: attrUsername, summary: "Conflicting credentials.", detail: "username conflicts with api_auth_email. Choose API token (api_auth_email + api_token) or basic (username + password), not both."},
		}
	}
	if rc.email == "" && rc.username == "" {
		return []validationErr{
			{attr: attrAPIAuthEmail, summary: "Missing credentials.", detail: "Provide api_auth_email with api_token for API token authentication, or set auth_method = \"basic\" and use username + password."},
			{attr: attrUsername, summary: "Missing credentials.", detail: "Provide username with password for basic authentication, or use api_auth_email + api_token for API token auth."},
		}
	}

	switch rc.authMethod {
	case "api_token":
		if rc.email == "" {
			errs = append(errs, validationErr{attr: attrAPIAuthEmail, summary: "Missing API Auth Email Configuration.", detail: "Provide 'api_auth_email' or set JIRA_API_EMAIL."})
		}
		if rc.apiToken == "" {
			errs = append(errs, validationErr{attr: attrAPIToken, summary: "Missing API Token Configuration.", detail: "Provide 'api_token' or set JIRA_API_TOKEN."})
		}
		if rc.username != "" {
			errs = append(errs, validationErr{attr: attrUsername, summary: "Attribute not allowed with api_token auth_method.", detail: "Remove 'username' (and 'password') or set auth_method = \"basic\"."})
		}
		if rc.password != "" {
			errs = append(errs, validationErr{attr: attrPassword, summary: "Attribute not allowed with api_token auth_method.", detail: "Remove 'password' (and 'username') or set auth_method = \"basic\"."})
		}
	case "basic":
		if rc.username == "" {
			errs = append(errs, validationErr{attr: attrUsername, summary: "Missing Username Configuration.", detail: "Provide 'username' or set JIRA_USERNAME."})
		}
		if rc.password == "" {
			errs = append(errs, validationErr{attr: attrPassword, summary: "Missing Password Configuration.", detail: "Provide 'password' or set JIRA_PASSWORD."})
		}
		if rc.email != "" {
			errs = append(errs, validationErr{attr: attrAPIAuthEmail, summary: "Attribute not allowed with basic auth_method.", detail: "Remove 'api_auth_email' (and 'api_token') or set auth_method = \"api_token\"."})
		}
		if rc.apiToken != "" {
			errs = append(errs, validationErr{attr: attrAPIToken, summary: "Attribute not allowed with basic auth_method.", detail: "Remove 'api_token' (and 'api_auth_email') or set auth_method = \"api_token\"."})
		}
	default:
		// already handled in validateBase, keep for completeness if validateBase is skipped
		errs = append(errs, validationErr{attr: attrAuthMethod, summary: "Invalid Auth Method Configuration.", detail: "auth_method must be 'api_token' or 'basic'."})
	}
	return errs
}

func validateResolvedConfig(rc resolvedConfig) []validationErr {
	var all []validationErr
	all = append(all, validateBase(rc)...)
	if len(all) == 0 { // if base fails, skip noisy follow-ups
		all = append(all, validateHTTP(rc)...)
		all = append(all, validateRetry(rc)...)
		all = append(all, validateAuth(rc)...)
	}

	// Before returning, sanitize any secrets from messages to prevent leakage.
	for i := range all {
		all[i] = sanitizeValidationError(all[i], rc)
	}
	return all
}
