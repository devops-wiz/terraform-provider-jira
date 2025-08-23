// SPDX-License-Identifier: MPL-2.0
//
// Test Sweeper for acceptance-test artifacts.
//
// Usage:
//   task sweep
// which runs:
//   go test ./internal/provider -test.v -args -sweep all
//
// Flags:
//   -sweep all|work_types|workflow_statuses[,csv]
//
// Notes:
// - Requires Jira Cloud env vars (see .junie/guidelines.md):
//     JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
//   (Basic auth also supported via JIRA_USERNAME/JIRA_PASSWORD.)
// - Operates only on known acceptance-test prefixes to avoid accidental deletion.
//   Work Types prefix:          "tf-acc-work-type"
//   Workflow Statuses prefix:   "tf-acc-workflow-status"

package provider

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

var sweepTargetsFlag string

func init() {
	if flag.CommandLine.Lookup("sweep") == nil {
		flag.StringVar(&sweepTargetsFlag, "sweep", "", "comma-separated targets to sweep: all, work_types, workflow_statuses")
	}
}

func TestMain(m *testing.M) {
	if sweepTargetsFlag != "" {
		// Run sweeper and exit without executing tests
		code := runSweep(sweepTargetsFlag)
		os.Exit(code)
	}
	os.Exit(m.Run())
}

const (
	accPrefixWorkType       = "tf-acc-work-type"
	accPrefixWorkflowStatus = "tf-acc-workflow-status"
)

// retry tuning for sweeper (kept conservative)
const (
	_sweepMaxAttempts   = 3
	_sweepBackoffBaseMS = 500
	_sweepBackoffMaxMS  = 5000
)

// test-local wrappers aligning to current helper names
func isContextError(err error) bool               { return IsContextError(err) }
func shouldRetry(status int, err error) bool      { return ShouldRetry(status, err) }
func parseRetryAfter(h http.Header) time.Duration { return ParseRetryAfter(h) }
func backoffDuration(attempt int) time.Duration {
	base := time.Duration(_sweepBackoffBaseMS) * time.Millisecond
	maxBackoff := time.Duration(_sweepBackoffMaxMS) * time.Millisecond
	return BackoffDuration(attempt, base, maxBackoff, 0.2)
}

func httpStatusAndHeaders(resp *models.ResponseScheme) (int, http.Header) {
	return HTTPStatusFromScheme(resp), responseHeadersFromScheme(resp)
}

//func sweepBackoff(attempt int) time.Duration {
//	base := time.Duration(_sweepBackoffBaseMS) * time.Millisecond
//	maxBackoff := time.Duration(_sweepBackoffMaxMS) * time.Millisecond
//	// Use a small jitter to avoid thundering herd during sweeps
//	return BackoffDuration(attempt, base, maxBackoff, 0.2)
//}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func classify(status int, err error, h http.Header) (category, hint string) {
	switch {
	case isContextError(err):
		return "canceled", "Operation canceled or deadline exceeded; adjust timeouts or check upstream cancellations."
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "permission", "Check Jira credentials/permissions for the configured account."
	case status == http.StatusNotFound:
		return "not_found", "Requested resource not found (404). It may already be deleted."
	case status == http.StatusTooManyRequests:
		ra := ParseRetryAfter(h)
		if ra > 0 {
			return "rate_limited", fmt.Sprintf("Rate limited (429). Server requested retry after %s.", ra)
		}
		return "rate_limited", "Rate limited (429). Retrying with backoff. Consider lowering Terraform parallelism or tuning retries."
	case status >= 500:
		return "server_error", "Server error (5xx). Retrying may succeed if transient."
	case status == 0 && err != nil:
		return "connectivity", "Connectivity or protocol error. Verify endpoint/network; transient errors will be retried."
	default:
		return "client_error", "Client error. Verify inputs and IDs."
	}
}

func runSweep(targets string) int {
	// Load env; no-op if not configured to avoid failing local runs/CI without secrets.
	endpoint := strings.TrimSpace(os.Getenv("JIRA_ENDPOINT"))
	email := strings.TrimSpace(os.Getenv("JIRA_API_EMAIL"))
	token := strings.TrimSpace(os.Getenv("JIRA_API_TOKEN"))

	username := strings.TrimSpace(os.Getenv("JIRA_USERNAME"))
	password := strings.TrimSpace(os.Getenv("JIRA_PASSWORD"))

	if endpoint == "" || ((email == "" || token == "") && (username == "" || password == "")) {
		fmt.Println("[sweeper] Jira environment not fully configured. Skipping sweep (no-op).")
		return 0
	}

	// Compose http client similar to provider defaults (30s timeout)
	httpClient := &http.Client{Timeout: 30 * time.Second}

	client, err := jira.New(httpClient, endpoint)
	if err != nil {
		fmt.Printf("[sweeper] Failed to create Jira client: %v\n", err)
		return 1
	}

	if email != "" && token != "" {
		client.Auth.SetBasicAuth(email, token)
	} else {
		client.Auth.SetBasicAuth(username, password)
	}
	client.Auth.SetUserAgent("devops-wiz/terraform-provider-jira/test-sweeper")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Validate connection (best-effort)
	if _, apiResp, err := client.MySelf.Details(ctx, nil); err != nil || HTTPStatusFromScheme(apiResp) >= 400 {
		fmt.Printf("[sweeper] Jira authentication check failed: status=%d err=%v\n", HTTPStatusFromScheme(apiResp), err)
		return 1
	}

	// Determine targets
	toRun := map[string]bool{}
	for _, t := range strings.Split(targets, ",") {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			continue
		}
		if t == "all" {
			toRun["work_types"] = true
			toRun["workflow_statuses"] = true
			break
		}
		toRun[t] = true
	}

	// Execute sweepers
	if toRun["work_types"] {
		sweepWorkTypes(ctx, client)
	}
	if toRun["workflow_statuses"] {
		sweepWorkflowStatuses(ctx, client)
	}

	fmt.Println("[sweeper] Sweep complete.")
	return 0
}

func sweepWorkTypes(ctx context.Context, client *jira.Client) {
	fmt.Println("[sweeper] Scanning work types...")
	var (
		items []*models.IssueTypeScheme
		resp  *models.ResponseScheme
		err   error
	)
	for attempt := 1; attempt <= _sweepMaxAttempts; attempt++ {
		items, resp, err = client.Issue.Type.Gets(ctx)
		status, headers := httpStatusAndHeaders(resp)
		if err == nil && status < 400 {
			break
		}
		cat, hint := classify(status, err, headers)
		fmt.Printf("[sweeper] Failed to list work types (attempt %d/%d): status=%d category=%s hint=%s err=%v\n", attempt, _sweepMaxAttempts, status, cat, RedactSecrets(hint), RedactSecrets(fmt.Sprint(err)))
		if !shouldRetry(status, err) || attempt == _sweepMaxAttempts {
			return
		}
		if d := parseRetryAfter(headers); d > 0 {
			if !sleepWithContext(ctx, d) {
				return
			}
			continue
		}
		if !sleepWithContext(ctx, backoffDuration(attempt)) {
			return
		}
	}
	// Sort by name for consistent output
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	var toDelete []string
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.Name), strings.ToLower(accPrefixWorkType)) {
			toDelete = append(toDelete, it.ID)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("[sweeper] No work types to delete.")
		return
	}

	fmt.Printf("[sweeper] Deleting %d work type(s) with prefix %q...\n", len(toDelete), accPrefixWorkType)
	for _, id := range toDelete {
		var delResp *models.ResponseScheme
		var delErr error
		for da := 1; da <= _sweepMaxAttempts; da++ {
			delResp, delErr = client.Issue.Type.Delete(ctx, id)
			ds, dh := httpStatusAndHeaders(delResp)
			if delErr == nil && (ds < 400 || ds == http.StatusNotFound) {
				if ds == http.StatusNotFound {
					fmt.Printf("[sweeper]   - work type id=%s already deleted (404)\n", id)
				} else {
					fmt.Printf("[sweeper]   - deleted work type id=%s\n", id)
				}
				break
			}
			cat, hint := classify(ds, delErr, dh)
			fmt.Printf("[sweeper]   - delete work type id=%s failed (attempt %d/%d): status=%d category=%s hint=%s err=%v\n", id, da, _sweepMaxAttempts, ds, cat, RedactSecrets(hint), RedactSecrets(fmt.Sprint(delErr)))
			if !shouldRetry(ds, delErr) || da == _sweepMaxAttempts {
				break
			}
			if d := parseRetryAfter(dh); d > 0 {
				if !sleepWithContext(ctx, d) {
					break
				}
				continue
			}
			if !sleepWithContext(ctx, backoffDuration(da)) {
				break
			}
		}
	}
}

func sweepWorkflowStatuses(ctx context.Context, client *jira.Client) {
	fmt.Println("[sweeper] Scanning workflow statuses...")
	for attempt := 1; attempt <= _sweepMaxAttempts; attempt++ {
		// Attempt to get all statuses. API typically supports listing all without IDs.
		statuses, resp, err := client.Workflow.Status.Gets(ctx, nil, nil)
		status, headers := httpStatusAndHeaders(resp)
		if err == nil && status < 400 {
			// Sort by name for consistent output
			sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })

			var toDelete []string
			for _, st := range statuses {
				if strings.HasPrefix(strings.ToLower(st.Name), strings.ToLower(accPrefixWorkflowStatus)) {
					toDelete = append(toDelete, st.ID)
				}
			}

			if len(toDelete) == 0 {
				fmt.Println("[sweeper] No workflow statuses to delete.")
				return
			}

			fmt.Printf("[sweeper] Deleting %d workflow status(es) with prefix %q...\n", len(toDelete), accPrefixWorkflowStatus)
			for _, id := range toDelete {
				var delResp *models.ResponseScheme
				var delErr error
				for da := 1; da <= _sweepMaxAttempts; da++ {
					delResp, delErr = client.Workflow.Status.Delete(ctx, []string{id})
					ds, dh := httpStatusAndHeaders(delResp)
					if delErr == nil && (ds < 400 || ds == http.StatusNotFound) {
						if ds == http.StatusNotFound {
							fmt.Printf("[sweeper]   - status id=%s already deleted (404)\n", id)
						} else {
							fmt.Printf("[sweeper]   - deleted status id=%s\n", id)
						}
						break
					}
					cat, hint := classify(ds, delErr, dh)
					fmt.Printf("[sweeper]   - delete status id=%s failed (attempt %d/%d): status=%d category=%s hint=%s err=%v\n", id, da, _sweepMaxAttempts, ds, cat, RedactSecrets(hint), RedactSecrets(fmt.Sprint(delErr)))
					if !shouldRetry(ds, delErr) || da == _sweepMaxAttempts {
						break
					}
					if d := parseRetryAfter(dh); d > 0 {
						if !sleepWithContext(ctx, d) {
							break
						}
						continue
					}
					if !sleepWithContext(ctx, backoffDuration(da)) {
						break
					}
				}
			}
			return
		}
		cat, hint := classify(status, err, headers)
		fmt.Printf("[sweeper] Failed to list workflow statuses (attempt %d/%d): status=%d category=%s hint=%s err=%v\n", attempt, _sweepMaxAttempts, status, cat, RedactSecrets(hint), RedactSecrets(fmt.Sprint(err)))
		if !shouldRetry(status, err) || attempt == _sweepMaxAttempts {
			return
		}
		if d := parseRetryAfter(headers); d > 0 {
			if !sleepWithContext(ctx, d) {
				return
			}
			continue
		}
		if !sleepWithContext(ctx, backoffDuration(attempt)) {
			return
		}
	}
}
