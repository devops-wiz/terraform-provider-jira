// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
)

func TestRedactJoin(t *testing.T) {
	parts := []string{"Authorization: Bearer abcdef123456", "contact: john.doe@example.com"}
	out := RedactJoin(parts, "; ")
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if strings.Contains(out, "Bearer abcdef123456") || strings.Contains(out, "john.doe@example.com") {
		t.Fatalf("expected redacted output, got: %s", out)
	}
	if !strings.Contains(out, "<redacted>") && !strings.Contains(out, "j***@example.com") {
		t.Fatalf("expected redaction markers in output, got: %s", out)
	}
}

func TestRedactJoin_SinglePart(t *testing.T) {
	parts := []string{"https://user:pass@example.com/path"}
	sep := " | "
	out := RedactJoin(parts, sep)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if strings.Contains(out, sep) {
		t.Fatalf("expected no separator to be added for single-part input; got: %q", out)
	}
	if strings.Contains(out, "user:pass@") {
		t.Fatalf("expected credentials to be redacted, got: %s", out)
	}
	if !strings.Contains(out, "<redacted>@") {
		t.Fatalf("expected redaction marker in URL credentials, got: %s", out)
	}
}

func TestRedactJoin_NoSensitiveData(t *testing.T) {
	parts := []string{"alpha", "beta", "gamma"}
	sep := ", "
	out := RedactJoin(parts, sep)
	want := strings.Join(parts, sep)
	if out != want {
		t.Fatalf("expected output equal to plain join when no sensitive data; got %q want %q", out, want)
	}
	if strings.Contains(out, "<redacted>") {
		t.Fatalf("did not expect any redaction markers in output; got: %s", out)
	}
}

func TestRedactSecrets_BasicCases(t *testing.T) {
	cases := []struct {
		in      string
		notHave []string
		haveAny []string
	}{
		{
			in:      "Authorization: Bearer abcdef123456",
			notHave: []string{"Bearer abcdef123456"},
			haveAny: []string{"Authorization: <redacted>", "<redacted>"},
		},
		{
			in:      "Authorization: Basic ZHVtbXk6cHdk",
			notHave: []string{"Basic ZHVtbXk6cHdk"},
			haveAny: []string{"Authorization: <redacted>"},
		},
		{
			in:      "Bearer secretToken12==",
			notHave: []string{"Bearer secretToken12=="},
			haveAny: []string{"Bearer <redacted>"},
		},
		{
			in:      "Basic QUJDREVGRw==",
			notHave: []string{"Basic QUJDREVGRw=="},
			haveAny: []string{"Basic <redacted>"},
		},
		{
			in:      "https://user:pass@example.com/path",
			notHave: []string{"user:pass@"},
			haveAny: []string{"https://<redacted>@example.com/path"},
		},
		{
			in:      "https://api.example.com?token=abc123&x=1",
			notHave: []string{"token=abc123"},
			haveAny: []string{"token=<redacted>"},
		},
		{
			in:      "api_token: abc12345",
			notHave: []string{"api_token: abc12345"},
			haveAny: []string{"api_token: <redacted>"},
		},
		{
			in:      "Contact: john.doe@example.com",
			notHave: []string{"john.doe@example.com"},
			haveAny: []string{"j***@example.com"},
		},
	}

	for i, c := range cases {
		out := RedactSecrets(c.in)
		for _, bad := range c.notHave {
			if strings.Contains(out, bad) {
				t.Fatalf("case %d: expected output to not contain %q, got: %s", i, bad, out)
			}
		}
		found := false
		for _, ok := range c.haveAny {
			if strings.Contains(out, ok) {
				found = true
				break
			}
		}
		if len(c.haveAny) > 0 && !found {
			t.Fatalf("case %d: expected output to contain one of %v, got: %s", i, c.haveAny, out)
		}
	}
}

func TestRedactHeaders(t *testing.T) {
	h := http.Header{
		"Authorization":     []string{"Bearer verysecrettoken"},
		"Cookie":            []string{"JSESSIONID=secret"},
		"X-Atlassian-Token": []string{"nocheck"},
		"X-Custom":          []string{"token=abc123"},
	}
	red := RedactHeaders(h)
	if got := red.Get("Authorization"); got != "<redacted>" {
		t.Fatalf("expected Authorization to be redacted, got: %q", got)
	}
	if got := red.Get("Cookie"); got != "<redacted>" {
		t.Fatalf("expected Cookie to be redacted, got: %q", got)
	}
	if got := red.Get("X-Atlassian-Token"); got != "<redacted>" {
		t.Fatalf("expected X-Atlassian-Token to be redacted, got: %q", got)
	}
	if got := red.Get("X-Custom"); !strings.Contains(got, "<redacted>") && strings.Contains(got, "abc123") {
		t.Fatalf("expected non-sensitive header values to be sanitized, got: %q", got)
	}
}

func TestErrorFrom_RedactsSensitiveError(t *testing.T) {
	var buf bytes.Buffer
	rs := &models.ResponseScheme{Code: 400, Response: &http.Response{StatusCode: 400, Header: http.Header{}}, Bytes: buf}
	sum, det := ErrorFromScheme(context.Background(), "op", rs, errors.New("authorization: Bearer abc123"))
	if strings.Contains(sum, "Bearer abc123") || strings.Contains(det, "Bearer abc123") {
		t.Fatalf("expected redaction of token in summary/detail; got sum=%q det=%q", sum, det)
	}
	if !strings.Contains(sum, "<redacted>") && !strings.Contains(det, "<redacted>") {
		t.Fatalf("expected <redacted> marker in redacted output; got sum=%q det=%q", sum, det)
	}
}

func TestRedactSecrets_URLQueryRedacted_NotStripped(t *testing.T) {
	in := "request failed for https://tenant.atlassian.net/rest/api/3/issue?token=abc123&foo=bar#frag"
	out := RedactSecrets(in)
	if !strings.Contains(out, "token=<redacted>") {
		t.Fatalf("expected token value redacted, got: %s", out)
	}
	if !strings.Contains(out, "foo=bar") {
		t.Fatalf("expected non-sensitive param preserved, got: %s", out)
	}
}

func TestRedactSecrets_JSONKeysRedacted(t *testing.T) {
	in := `{"access_token":"AAA","refresh_token":"BBB","client_secret":"CCC","password":"DDD","authorization":"Bearer EEE"}`
	out := RedactSecrets(in)
	for _, k := range []string{"access_token", "refresh_token", "client_secret", "password", "authorization"} {
		if strings.Contains(out, k+"\":\"AAA") || strings.Contains(out, k+"\":\"BBB") || strings.Contains(out, k+"\":\"CCC") || strings.Contains(out, k+"\":\"DDD") || strings.Contains(out, k+"\":\"EEE") {
			t.Fatalf("expected %s value redacted, got: %s", k, out)
		}
	}
	if !strings.Contains(out, "\"access_token\":\"<redacted>\"") {
		t.Fatalf("expected access_token redacted, got: %s", out)
	}

	in2 := `{'access_token':'AAA','refresh_token':'BBB','client_secret':'CCC','password':'DDD','authorization':'Basic ZZZ'}`
	out2 := RedactSecrets(in2)
	if strings.Contains(out2, "AAA") || strings.Contains(out2, "BBB") || strings.Contains(out2, "CCC") || strings.Contains(out2, "DDD") || strings.Contains(out2, "ZZZ") {
		t.Fatalf("expected all secret values redacted in single-quoted JSON-ish, got: %s", out2)
	}
}

func TestRedactSecrets_AuthorizationKV_And_XApiKey(t *testing.T) {
	in := "authorization=Bearer abcdef123 xyz; X-Api-Key: supersecret; X-Api-Token: tok"
	out := RedactSecrets(in)
	if strings.Contains(out, "Bearer abcdef123") || strings.Contains(out, "supersecret") || strings.Contains(out, "X-Api-Token: tok") {
		t.Fatalf("expected authorization/X-Api-* redacted, got: %s", out)
	}
	if !strings.Contains(out, "authorization=<redacted>") || !strings.Contains(out, "X-Api-Key: <redacted>") {
		t.Fatalf("expected redaction markers present, got: %s", out)
	}
}

func TestRedactSecrets_CookieAndSetCookie_Redacted(t *testing.T) {
	in := "Cookie: JSESSIONID=abc; Path=/; HttpOnly\nSet-Cookie: other=val"
	out := RedactSecrets(in)
	if !strings.Contains(out, "Cookie: <redacted>") || !strings.Contains(out, "Set-Cookie: <redacted>") {
		t.Fatalf("expected Cookie and Set-Cookie redacted, got: %s", out)
	}
}

func TestRedactSecrets_Idempotent(t *testing.T) {
	in := "Authorization: Bearer TOPSECRET"
	once := RedactSecrets(in)
	twice := RedactSecrets(once)
	if once != twice {
		t.Fatalf("expected idempotent redaction, got first=%q second=%q", once, twice)
	}
}

func TestHTTPStatus_BasicCases(t *testing.T) {
	if got := HTTPStatusFromScheme(testhelpers.MkRS(201, nil, "")); got != 201 {
		t.Fatalf("expected 201, got %d", got)
	}
	if got := HTTPStatusFromScheme(testhelpers.MkRS(204, nil, "")); got != 204 {
		t.Fatalf("expected 204, got %d", got)
	}
	// nil response
	if got := HTTPStatusFromScheme(nil); got != 0 {
		t.Fatalf("expected 0 for nil provider, got %d", got)
	}
}

func TestIsSuccess(t *testing.T) {
	cases := []struct {
		code int
		exp  bool
	}{
		{199, false}, {200, true}, {204, true}, {299, true}, {300, false},
	}
	for _, c := range cases {
		if got := IsSuccess(c.code); got != c.exp {
			t.Fatalf("IsSuccess(%d)=%v, want %v", c.code, got, c.exp)
		}
	}
}

func TestErrorFromWithOptions_HeadersAndBody(t *testing.T) {
	hs := http.Header{
		"Retry-After":           []string{"30"},
		"X-Request-Id":          []string{"req-123"},
		"X-RateLimit-Remaining": []string{"49"},
	}
	rs := testhelpers.MkRS(400, hs, "{\"message\":\"error\"}")
	_, detail := ErrorFromSchemeWithOptions(context.Background(), "test op", rs, nil, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true})
	if !strings.Contains(detail, "Headers:") {
		t.Fatalf("expected Headers to be included in detail: %s", detail)
	}
	if !strings.Contains(detail, "Retry-After=30") || !strings.Contains(detail, "X-Request-Id=req-123") {
		t.Fatalf("expected header hints present in detail: %s", detail)
	}
	if !strings.Contains(detail, "Response snippet:") {
		t.Fatalf("expected Response snippet in detail: %s", detail)
	}
}

func TestResponseDebugInfo_HTTPHeader(t *testing.T) {
	rs := testhelpers.MkRS(500, http.Header{"X-RateLimit-Window": []string{"60"}}, "internal error")
	body, headers := responseDebugInfoFromScheme(rs, 1024)
	if body == "" {
		t.Fatalf("expected non-empty body snippet")
	}
	found := false
	for _, h := range headers {
		if strings.Contains(h, "X-RateLimit-Window=60") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected X-RateLimit-Window header hint, got: %v", headers)
	}
}

func TestErrorFrom_ContextHints(t *testing.T) {
	rs := testhelpers.MkRS(0, nil, "")
	_, d1 := ErrorFromScheme(context.Background(), "test op", rs, context.DeadlineExceeded)
	if !strings.Contains(d1, "deadline exceeded") {
		t.Fatalf("expected deadline exceeded hint, got: %s", d1)
	}
	_, d2 := ErrorFromScheme(context.Background(), "test op", rs, context.Canceled)
	if !strings.Contains(d2, "canceled") {
		t.Fatalf("expected canceled hint, got: %s", d2)
	}
}

func TestEnsureSuccessOrDiag_BasicSuccess(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromScheme(context.Background(), "create", testhelpers.MkRS(200, nil, ""), nil, &diags)
	if !ok {
		t.Fatalf("expected ok=true for 200")
	}
	if diags.HasError() {
		t.Fatalf("expected no diagnostics errors")
	}
}

func TestEnsureSuccessOrDiag_AcceptableStatus(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromSchemeWithOptions(context.Background(), "update", testhelpers.MkRS(204, nil, ""), nil, &diags, &EnsureSuccessOrDiagOptions{AcceptableStatuses: []int{204}})
	if !ok || diags.HasError() {
		t.Fatalf("expected ok for acceptable 204 and no diagnostics")
	}
}

func TestEnsureSuccessOrDiag_Read404TreatedAsNotFound(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromSchemeWithOptions(context.Background(), "read", testhelpers.MkRS(404, nil, ""), nil, &diags, &EnsureSuccessOrDiagOptions{TreatRead404AsNotFound: true})
	if !ok || diags.HasError() {
		t.Fatalf("expected ok for read 404 treated as not found")
	}
}

func TestEnsureSuccessOrDiag_Delete404TreatedAsSuccess(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromSchemeWithOptions(context.Background(), "delete", testhelpers.MkRS(404, nil, ""), nil, &diags, &EnsureSuccessOrDiagOptions{TreatDelete404AsSuccess: true})
	if !ok || diags.HasError() {
		t.Fatalf("expected ok for delete 404 treated as success")
	}
}

func TestEnsureSuccessOrDiag_ClientErrorProducesDiag(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromScheme(context.Background(), "op", testhelpers.MkRS(400, nil, ""), nil, &diags)
	if ok {
		t.Fatalf("expected ok=false for 400 status")
	}
	if !diags.HasError() {
		t.Fatalf("expected diagnostics to have an error for 400 status")
	}
}

func TestEnsureSuccessOrDiag_ErrOverridesSuccess(t *testing.T) {
	var diags diag.Diagnostics
	ok := EnsureSuccessOrDiagFromScheme(context.Background(), "op", testhelpers.MkRS(200, nil, ""), context.DeadlineExceeded, &diags)
	if ok {
		t.Fatalf("expected ok=false when err is set even with 200 status")
	}
	if !diags.HasError() {
		t.Fatalf("expected diagnostics to have an error when err is set")
	}
}

func TestErrorFrom_StatusParseFailure_NoteAndZero(t *testing.T) {
	// No status set (0) and no embedded http.Response provided -> parse failure
	rs := &models.ResponseScheme{}
	_, detail := ErrorFromScheme(context.Background(), "op", rs, nil)
	if !strings.Contains(detail, "Note: Could not parse HTTP status") {
		t.Fatalf("expected parse Note in detail, got: %s", detail)
	}
	if !strings.Contains(detail, "HTTP status: 0") {
		t.Fatalf("expected HTTP status: 0 in detail, got: %s", detail)
	}
}

func TestErrorFrom_OutOfRange_StatusNote(t *testing.T) {
	rs := testhelpers.MkRS(99, nil, "")
	_, detail := ErrorFromScheme(context.Background(), "op", rs, nil)
	if !strings.Contains(detail, "Note: Could not parse HTTP status") {
		t.Fatalf("expected parse Note for out-of-range, got: %s", detail)
	}
	if !strings.Contains(detail, "HTTP status: 0") {
		t.Fatalf("expected HTTP status: 0 for out-of-range, got: %s", detail)
	}
}

func TestErrorFromWithOptions_LargeBodyTruncation_And_Redaction(t *testing.T) {
	body := testhelpers.BuildLargeBody()
	hs := http.Header{
		"Retry-After":  []string{"30"},
		"X-Request-Id": []string{"req-abc"},
	}
	rs := testhelpers.MkRSWithBodyAndHeaders(http.StatusTooManyRequests, hs, body)
	maxBodyBytes := 1024
	_, detail := ErrorFromSchemeWithOptions(context.Background(), "create", rs, nil, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true, MaxBodyBytes: maxBodyBytes})
	assert.Contains(t, detail, "Headers: Retry-After=30; X-Request-Id=req-abc", "expected header hints in detail, got: %s", detail)
	assert.Contains(t, detail, "Response snippet:", "expected Response snippet in detail, got: %s", detail)
	// Ensure truncation roughly obeys maxBodyBytes (allow some overhead for prefix text in detail)
	idx := strings.Index(detail, "Response snippet: ")
	if idx == -1 {
		t.Fatalf("missing response snippet")
	}
	snippet := detail[idx+len("Response snippet: "):]
	if len(snippet) > maxBodyBytes+32 { // +32 for potential ellipsis and formatting slack
		t.Fatalf("expected truncated snippet, len=%d exceeds limit", len(snippet))
	}
	// Ensure secrets are redacted
	if strings.Contains(snippet, "TOPSECRET") || strings.Contains(snippet, "PWD") || strings.Contains(snippet, "AAA") {
		t.Fatalf("expected secrets redacted in snippet; got: %s", snippet)
	}
	if !strings.Contains(snippet, "<redacted>") {
		t.Fatalf("expected redaction markers in snippet; got: %s", snippet)
	}
}

func TestResponseDebugInfo_HeaderAllowlist(t *testing.T) {
	// Create many headers, but only allowlist should be surfaced.
	hs := http.Header{}
	for i := 0; i < 1500; i++ {
		hs.Set("X-Noise-"+strconv.Itoa(i), "v")
	}
	hs.Set("Retry-After", "7")
	hs.Set("X-Request-Id", "req-999")
	rs := testhelpers.MkRSWithBodyAndHeaders(http.StatusServiceUnavailable, hs, "srv err")
	_, hints := responseDebugInfoFromScheme(rs, 256)
	joined := strings.Join(hints, "; ")
	if !strings.Contains(joined, "Retry-After=7") || !strings.Contains(joined, "X-Request-Id=req-999") {
		t.Fatalf("expected allowlisted headers present; got: %s", joined)
	}
	if strings.Contains(joined, "X-Noise-") {
		t.Fatalf("unexpected noise headers leaked into hints: %s", joined)
	}
}

func TestShouldRetry_DNSError_NoTimeout_NoRetry(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "invalid.local" /* IsTimeout false by default */}
	if ShouldRetry(0, err) {
		t.Fatalf("expected no retry for non-timeout DNS error: %v", err)
	}
}

func TestShouldRetry_UnexpectedEOF_Retry(t *testing.T) {
	if !ShouldRetry(0, io.ErrUnexpectedEOF) {
		t.Fatalf("expected retry for io.ErrUnexpectedEOF")
	}
}

func TestShouldRetry_ConnReset_Retry(t *testing.T) {
	// Wrap ECONNRESET in typical layers to simulate transport error
	opErr := &net.OpError{Err: &os.SyscallError{Syscall: "read", Err: syscall.ECONNRESET}}
	if !ShouldRetry(0, opErr) {
		t.Fatalf("expected retry for connection reset error: %v", opErr)
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	h := http.Header{"Retry-After": []string{"30"}}
	d := ParseRetryAfter(h)
	if d < 30*time.Second || d > 31*time.Second {
		t.Fatalf("expected ~30s, got %s", d)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(3 * time.Second).UTC().Format(http.TimeFormat)
	h := http.Header{"Retry-After": []string{future}}
	d := ParseRetryAfter(h)
	if d <= 0 {
		t.Fatalf("expected positive duration for HTTP-date, got %s", d)
	}
	if d > 10*time.Second { // generous upper bound to avoid flakes
		t.Fatalf("unexpectedly large duration: %s", d)
	}
}

func TestParseRetryAfter_InvalidOrMissing(t *testing.T) {
	if got := ParseRetryAfter(http.Header{}); got != 0 {
		t.Fatalf("expected 0 for missing header, got %s", got)
	}
	h := http.Header{"Retry-After": []string{"not-a-number"}}
	if got := ParseRetryAfter(h); got != 0 {
		t.Fatalf("expected 0 for invalid header, got %s", got)
	}
}

func TestBackoffDuration_NoJitter_Cap(t *testing.T) {
	base := 500 * time.Millisecond
	maxBackoff := 5 * time.Second
	cases := []struct {
		attempt int
		exp     time.Duration
	}{
		{1, 500 * time.Millisecond},
		{2, 1 * time.Second},
		{3, 2 * time.Second},
		{4, 4 * time.Second},
		{5, 5 * time.Second},  // capped
		{10, 5 * time.Second}, // capped
	}
	for _, c := range cases {
		got := BackoffDuration(c.attempt, base, maxBackoff, 0)
		if got != c.exp {
			t.Fatalf("attempt %d: got %s, want %s", c.attempt, got, c.exp)
		}
	}
}

func TestBackoffDuration_WithJitter_Bounds(t *testing.T) {
	base := 500 * time.Millisecond
	maxBackoff := 5 * time.Second
	attempt := 4 // expected base*2^(3) = 4s
	exp := 4 * time.Second
	j := 0.2
	lower := time.Duration(float64(exp) * (1 - j))
	upper := time.Duration(float64(exp) * (1 + j))
	if upper > maxBackoff {
		upper = maxBackoff
	}
	for i := 0; i < 20; i++ {
		got := BackoffDuration(attempt, base, maxBackoff, j)
		if got < lower || got > upper {
			t.Fatalf("jittered duration out of bounds: got=%s lower=%s upper=%s", got, lower, upper)
		}
	}
}

func TestShouldRetry_Cases(t *testing.T) {
	if !ShouldRetry(http.StatusTooManyRequests, nil) {
		t.Fatal("expected retry for 429")
	}
	if !ShouldRetry(http.StatusBadGateway, nil) {
		t.Fatal("expected retry for 502")
	}
	if ShouldRetry(http.StatusUnauthorized, nil) {
		t.Fatal("expected no retry for 401")
	}
	if ShouldRetry(http.StatusForbidden, nil) {
		t.Fatal("expected no retry for 403")
	}
	if ShouldRetry(0, context.DeadlineExceeded) {
		t.Fatal("expected no retry for context deadline exceeded")
	}
	if !ShouldRetry(0, testhelpers.NewFakeNetErr(true)) {
		t.Fatal("expected retry for net.Error timeout with status 0")
	}
}

func TestIsContextError(t *testing.T) {
	if !IsContextError(context.Canceled) || !IsContextError(context.DeadlineExceeded) {
		t.Fatal("expected IsContextError to be true for context errors")
	}
	if IsContextError(nil) {
		t.Fatal("expected IsContextError(false) for nil error")
	}
}
