// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Minimal testAccPreCheck for acceptance tests; requires API token auth envs and lead account.
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("JIRA_ENDPOINT"); v == "" {
		t.Fatal("JIRA_ENDPOINT must be set for acceptance tests")
	} else {
		u, err := url.Parse(v)
		if err != nil {
			t.Fatalf("JIRA_ENDPOINT is not a valid URL: %v", err)
		}
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
		if strings.Contains(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.") {
			t.Fatal("JIRA_ENDPOINT must not point to localhost")
		}
	}

	if v := os.Getenv("JIRA_API_EMAIL"); v == "" {
		t.Fatal("JIRA_API_EMAIL must be set for acceptance tests")
	} else {
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

	// Require a project lead account ID for jira_project acceptance tests
	if v := os.Getenv("JIRA_PROJECT_TEST_ROLE_LEAD_ID"); v == "" {
		t.Fatal("JIRA_PROJECT_TEST_ROLE_LEAD_ID must be set for acceptance tests involving jira_project")
	} else {
		if strings.ContainsAny(v, " \t\r\n") {
			t.Fatal("JIRA_PROJECT_TEST_ROLE_LEAD_ID must not contain whitespace")
		}
		if len(v) < 5 {
			t.Fatal("JIRA_PROJECT_TEST_ROLE_LEAD_ID appears too short")
		}
	}
}

// Provider factory for acceptance tests
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"jira": providerserver.NewProtocol6WithError(New("test")()),
}
