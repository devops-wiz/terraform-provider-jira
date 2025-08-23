// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

// Centralized attribute names used in provider configuration schema and validation
const (
	attrEndpoint            = "endpoint"
	attrAuthMethod          = "auth_method"
	attrAPIToken            = "api_token"
	attrAPIAuthEmail        = "api_auth_email"
	attrUsername            = "username"
	attrPassword            = "password"
	attrHTTPTimeoutSeconds  = "http_timeout_seconds"
	attrRetryOn4295xx       = "retry_on_429_5xx"
	attrRetryMaxAttempts    = "retry_max_attempts"
	attrRetryInitialBackoff = "retry_initial_backoff_ms"
	attrRetryMaxBackoff     = "retry_max_backoff_ms"
	attrEmailRedactionMode  = "email_redaction_mode"
)

// Centralized provider defaults
const (
	defaultAuthMethod            = "api_token"
	defaultHTTPTimeoutSeconds    = 30
	defaultRetryOn4295xx         = true
	defaultRetryMaxAttempts      = 4
	defaultRetryInitialBackoffMs = 500
	defaultRetryMaxBackoffMs     = 5000
	defaultEmailRedactionMode    = "full"
)
