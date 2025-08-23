// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// buildHTTPClient constructs the HTTP client with optional retry/backoff policy.
func buildHTTPClient(rc resolvedConfig) *http.Client {
	if rc.retryOn4295xx {
		rcClient := retryablehttp.NewClient()
		rcClient.RetryMax = rc.retryMaxAttempts
		rcClient.RetryWaitMin = time.Duration(rc.retryInitialBackoffMs) * time.Millisecond
		rcClient.RetryWaitMax = time.Duration(rc.retryMaxBackoffMs) * time.Millisecond
		// keep default CheckRetry (retries on 429/5xx and honors Retry-After)
		// disable noisy logging unless debug is desired
		rcClient.Logger = nil
		httpClient := rcClient.StandardClient()
		httpClient.Timeout = time.Duration(rc.httpTimeoutSeconds) * time.Second
		return httpClient
	}
	return &http.Client{Timeout: time.Duration(rc.httpTimeoutSeconds) * time.Second}
}

// initJiraClient creates the Jira client, sets authentication and user agent.
func (j *JiraProvider) initJiraClient(httpClient *http.Client, rc resolvedConfig) (*jira.Client, error) {
	client, err := jira.New(httpClient, rc.endpoint)
	if err != nil {
		return nil, err
	}

	switch rc.authMethod {
	case "api_token":
		client.Auth.SetBasicAuth(rc.email, rc.apiToken)
	case "basic":
		client.Auth.SetBasicAuth(rc.username, rc.password)
	default:
		// Should be validated earlier; return an explicit error if reached.
		return nil, fmt.Errorf("invalid auth_method %q", rc.authMethod)
	}

	client.Auth.SetUserAgent(fmt.Sprintf("devops-wiz/terraform-provider-jira/%s", j.version))
	return client, nil
}

// testConnection checks API connectivity and appends diagnostics on failure.
func (j *JiraProvider) testConnection(ctx context.Context, client *jira.Client, diags *diag.Diagnostics) bool {
	_, apiResp, err := client.MySelf.Details(ctx, nil)
	return EnsureSuccessOrDiagFromScheme(ctx, "authenticate (myself)", apiResp, err, diags)
}
