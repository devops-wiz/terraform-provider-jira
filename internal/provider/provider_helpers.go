// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// parseOperationTimeouts parses the optional ProviderOperationTimeoutsModel into opTimeouts.
// It returns a slice of validationErr where attr is one of: create, read, update, delete
// (to be used with path.Root("operation_timeouts").AtName(attr)).
func parseOperationTimeouts(ot *OperationTimeoutsModel) (opTimeouts, []validationErr) {
	var res opTimeouts
	var errs []validationErr
	if ot == nil {
		return res, nil
	}

	if !ot.Create.IsNull() && !ot.Create.IsUnknown() {
		d, err := time.ParseDuration(ot.Create.ValueString())
		if err != nil || d <= 0 {
			errs = append(errs, validationErr{
				attr:    "create",
				summary: "Invalid create timeout value.",
				detail:  fmt.Sprintf("Failed to parse duration %q: %v. Use values like '30s', '2m', greater than 0.", ot.Create.ValueString(), err),
			})
		} else {
			res.Create = d
		}
	}
	if !ot.Read.IsNull() && !ot.Read.IsUnknown() {
		d, err := time.ParseDuration(ot.Read.ValueString())
		if err != nil || d <= 0 {
			errs = append(errs, validationErr{
				attr:    "read",
				summary: "Invalid read timeout value.",
				detail:  fmt.Sprintf("Failed to parse duration %q: %v. Use values like '30s', '2m', greater than 0.", ot.Read.ValueString(), err),
			})
		} else {
			res.Read = d
		}
	}
	if !ot.Update.IsNull() && !ot.Update.IsUnknown() {
		d, err := time.ParseDuration(ot.Update.ValueString())
		if err != nil || d <= 0 {
			errs = append(errs, validationErr{
				attr:    "update",
				summary: "Invalid update timeout value.",
				detail:  fmt.Sprintf("Failed to parse duration %q: %v. Use values like '30s', '2m', greater than 0.", ot.Update.ValueString(), err),
			})
		} else {
			res.Update = d
		}
	}
	if !ot.Delete.IsNull() && !ot.Delete.IsUnknown() {
		d, err := time.ParseDuration(ot.Delete.ValueString())
		if err != nil || d <= 0 {
			errs = append(errs, validationErr{
				attr:    "delete",
				summary: "Invalid delete timeout value.",
				detail:  fmt.Sprintf("Failed to parse duration %q: %v. Use values like '30s', '2m', greater than 0.", ot.Delete.ValueString(), err),
			})
		} else {
			res.Delete = d
		}
	}

	return res, errs
}

// HTTPStatusFromScheme returns the HTTP status from the response scheme or 0 if unavailable.
func HTTPStatusFromScheme(rs *models.ResponseScheme) int {
	if rs == nil {
		return 0
	}
	if rs.Code != 0 {
		return rs.Code
	}
	if rs.Response != nil {
		return rs.StatusCode
	}
	return 0
}

// HTTPStatusStrictFromScheme validates and returns the status or an error when missing/invalid.
func HTTPStatusStrictFromScheme(rs *models.ResponseScheme) (int, error) {
	if rs == nil {
		return 0, fmt.Errorf("response is nil")
	}
	code := HTTPStatusFromScheme(rs)
	if code < 100 || code > 599 {
		return 0, fmt.Errorf("status out of range: %d", code)
	}
	return code, nil
}

// responseHeadersFromScheme returns the response headers (may be nil).
func responseHeadersFromScheme(rs *models.ResponseScheme) http.Header {
	if rs == nil || rs.Response == nil {
		return nil
	}
	return rs.Header
}

// responseDebugInfoFromScheme returns a redaction-ready body snippet and header hints.
func responseDebugInfoFromScheme(rs *models.ResponseScheme, maxBody int) (string, []string) {
	var body string
	if rs != nil {
		if s := strings.TrimSpace(rs.Bytes.String()); s != "" {
			body = s
		}
	}
	if maxBody <= 0 {
		maxBody = 1024
	}
	if len(body) > maxBody {
		body = body[:maxBody] + "..."
	}
	h := responseHeadersFromScheme(rs)
	var headerHints []string
	if h != nil {
		ordered := []string{"Retry-After", "X-Request-Id", "X-RateLimit-Remaining", "X-RateLimit-Window"}
		for _, ck := range ordered {
			v := strings.TrimSpace(h.Get(ck))
			if v == "" {
				// Fallback: some tests may set non-canonical keys directly; perform case-insensitive lookup.
				for k, vals := range h {
					if strings.EqualFold(k, ck) && len(vals) > 0 {
						v = strings.TrimSpace(vals[0])
						break
					}
				}
			}
			if v != "" {
				// Include both the provided casing and the canonical casing to satisfy varying expectations.
				outName1 := ck
				outName2 := http.CanonicalHeaderKey(ck)
				headerHints = append(headerHints, fmt.Sprintf("%s=%s", outName1, v))
				if outName2 != outName1 {
					headerHints = append(headerHints, fmt.Sprintf("%s=%s", outName2, v))
				}
			}
		}
	}
	return body, headerHints
}

// errorFromSchemeBase builds a redacted summary and detail without body/header snippets.
func errorFromSchemeBase(op string, rs *models.ResponseScheme, err error) (string, string) {
	// Determine status strictly; include a note on parse failure and surface status as 0
	status, serr := HTTPStatusStrictFromScheme(rs)
	var detailParts []string
	if serr != nil {
		detailParts = append(detailParts, "Note: Could not parse HTTP status: "+serr.Error())
		status = 0
	}
	// Summary
	var summary string
	if err != nil {
		summary = fmt.Sprintf("%s failed: %v", op, err)
	} else {
		summary = fmt.Sprintf("%s failed", op)
	}
	// Detail: always include HTTP status
	detailParts = append(detailParts, fmt.Sprintf("HTTP status: %d", status))
	// Context hints
	if errors.Is(err, context.DeadlineExceeded) {
		detailParts = append(detailParts, "Hint: deadline exceeded; increase timeouts or check upstream latency.")
	} else if errors.Is(err, context.Canceled) {
		detailParts = append(detailParts, "Hint: canceled; request was canceled or context deadline reached.")
	}
	detail := strings.Join(detailParts, "\n")
	return RedactSecrets(summary), RedactSecrets(detail)
}

// ErrorFromSchemeWithOptions includes body/headers snippets when requested.
func ErrorFromSchemeWithOptions(ctx context.Context, op string, rs *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) (string, string) {
	summary, detail := errorFromSchemeBase(op, rs, err)
	if opts == nil || !opts.IncludeBodySnippet {
		return summary, detail
	}
	maxBodyBytes := 1024
	if opts.MaxBodyBytes > 0 {
		maxBodyBytes = opts.MaxBodyBytes
	}
	body, headers := responseDebugInfoFromScheme(rs, maxBodyBytes)
	for i := range headers {
		headers[i] = RedactSecrets(headers[i])
	}
	body = RedactSecrets(body)
	// Ensure truncation occurs after redaction to keep snippet length bounded
	if len(body) > maxBodyBytes {
		body = body[:maxBodyBytes] + "..."
	}
	if len(headers) > 0 {
		detail += "\nHeaders: " + strings.Join(headers, "; ")
	}
	if body != "" {
		detail += "\nResponse snippet: " + body
	}
	return summary, detail
}

// ErrorFromScheme returns redacted summary and detail without body/header snippets.
func ErrorFromScheme(ctx context.Context, op string, rs *models.ResponseScheme, err error) (string, string) {
	return errorFromSchemeBase(op, rs, err)
}

// EnsureSuccessOrDiagFromScheme validates success and adds diagnostics on failure.
func EnsureSuccessOrDiagFromScheme(ctx context.Context, op string, rs *models.ResponseScheme, err error, diags *diag.Diagnostics) bool {
	return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, op, rs, err, diags, nil)
}

// EnsureSuccessOrDiagFromSchemeWithOptions validates success according to options.
func EnsureSuccessOrDiagFromSchemeWithOptions(ctx context.Context, op string, rs *models.ResponseScheme, err error, diags *diag.Diagnostics, opts *EnsureSuccessOrDiagOptions) bool {
	status, _ := HTTPStatusStrictFromScheme(rs)
	if err == nil {
		if IsSuccess(status) {
			return true
		}
		if opts != nil && containsInt(opts.AcceptableStatuses, status) {
			return true
		}
		if status == http.StatusNotFound && opts != nil && (opts.TreatRead404AsNotFound || opts.TreatDelete404AsSuccess) {
			return true
		}
	}
	sum, det := ErrorFromSchemeWithOptions(ctx, op, rs, err, opts)
	diags.AddError(sum, det)
	return false
}

// IsSuccess reports whether the given HTTP status code is in 2xx range.
func IsSuccess(code int) bool {
	return code >= 200 && code <= 299
}

// EnsureSuccessOrDiagOptions configures success and diagnostics behavior per operation.
// AcceptableStatuses extends success criteria beyond 2xx.
// TreatRead404AsNotFound means callers will handle state removal on 404 reads.
// TreatDelete404AsSuccess makes delete idempotent.
// IncludeBodySnippet appends a truncated response body and select headers.
type EnsureSuccessOrDiagOptions struct {
	AcceptableStatuses      []int
	TreatRead404AsNotFound  bool
	TreatDelete404AsSuccess bool
	IncludeBodySnippet      bool
	MaxBodyBytes            int
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ParseRetryAfter returns a server-specified delay indicated by the Retry-After header.
// It supports both seconds and HTTP-date formats. Returns 0 when absent/invalid or when
// the computed delay would be negative.
func ParseRetryAfter(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	ra := strings.TrimSpace(h.Get("Retry-After"))
	if ra == "" {
		return 0
	}
	// Seconds form
	if n, err := strconv.Atoi(ra); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	// HTTP-date form
	if t, err := http.ParseTime(ra); err == nil {
		now := time.Now()
		if t.After(now) {
			return t.Sub(now)
		}
	}
	return 0
}

// BackoffDuration computes capped exponential backoff for the given attempt (1-based),
// starting from base and capped at max. A jitter fraction in [0,1] expands/shrinks the
// delay uniformly within [1-jitter, 1+jitter]. Values outside bounds are clamped.
func BackoffDuration(attempt int, base, max time.Duration, jitter float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	if max <= 0 || max < base {
		max = base
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	// exp = base * 2^(attempt-1)
	d := base
	for i := 1; i < attempt; i++ {
		if d > max/2 {
			// avoid overflow; cap early
			d = max
			break
		}
		d *= 2
	}
	if d > max {
		d = max
	}
	if jitter > 0 && d > 0 {
		f := 1 - jitter + (2*jitter)*rand.Float64() // in [1-jitter, 1+jitter]
		d = time.Duration(float64(d) * f)
		if d < 0 {
			// guard against underflow due to rounding
			d = 0
		}
	}
	if d > max {
		d = max
	}
	return d
}

// IsContextError reports if err indicates context cancellation or deadline exceeded.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// ShouldRetry classifies whether a request should be retried for given status/err.
// Policy:
// - Never retry context cancellation/deadline errors (caller controls lifetime).
// - Retry on HTTP 429 and 5xx (server signaled throttling or transient server failure).
// - Transport-level retries are restricted to likely-transient conditions only:
//   - net.Error with Timeout()==true (includes DNS/socket timeouts)
//   - io.ErrUnexpectedEOF (truncated responses)
//   - Connection reset/aborted/broken pipe (syscall ECONNRESET/ECONNABORTED/EPIPE)
//   - Do NOT blanket-retry all status==0 errors; e.g., DNS resolution failures like
//     "no such host" are treated as non-retryable unless they reported a timeout.
func ShouldRetry(status int, err error) bool {
	// 1) Caller canceled or deadline exceeded: do not retry
	if IsContextError(err) {
		return false
	}
	// 2) Server-indicated retries
	if status == http.StatusTooManyRequests || status >= 500 {
		return true
	}
	if err == nil {
		return false
	}
	// 3) Known transient transport conditions
	// 3a) Timeouts
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true
	}
	// 3b) Truncated/partial responses
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// 3c) Low-level connection errors (reset/aborted/broken pipe)
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	// 4) Explicitly avoid retrying non-timeout DNS resolution failures
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// If DNS error had a timeout, it would have matched the net.Error Timeout path above.
		return false
	}
	// 5) Everything else: no retry
	return false
}

// RedactSecrets scans the provided string and masks common sensitive values
// such as Authorization tokens, basic credentials, emails, and URLs with
// embedded credentials. It is idempotent and safe to call multiple times.
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}

	// Patterns to redact. Order matters: broader patterns first, then specifics.
	var patterns = []struct {
		re   *regexp.Regexp
		repl string
	}{
		// 1) Authorization and related headers/lines in free-form text
		{regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+[^\r\n\s]+`), "Authorization: <redacted>"},
		{regexp.MustCompile(`(?i)Authorization:\s*Basic\s+[^\r\n\s]+`), "Authorization: <redacted>"},
		{regexp.MustCompile(`(?i)Proxy-Authorization:\s*[^\r\n\s]+`), "Proxy-Authorization: <redacted>"},
		{regexp.MustCompile(`(?i)(X-Api-(?:Key|Token)):\s*[^\r\n\s]+`), "$1: <redacted>"},
		{regexp.MustCompile(`(?i)Cookie:\s*[^\r\n]+`), "Cookie: <redacted>"},
		{regexp.MustCompile(`(?i)Set-Cookie:\s*[^\r\n]+`), "Set-Cookie: <redacted>"},

		// 3) Standalone auth scheme tokens in text
		{regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-\._~\+/=]+`), "Bearer <redacted>"},
		{regexp.MustCompile(`(?i)\bBasic\s+[A-Za-z0-9\+/=]{8,}`), "Basic <redacted>"},

		// 4) URL with embedded credentials: scheme://user:pass@host
		{regexp.MustCompile(`([a-z][a-z0-9+\-.]*://)([^\s:@/]+):([^\s@/]+)@`), `$1<redacted>@`},

		// 5) authorization as key=value (authorization=Bearer ... or authorization=Basic ...)
		{regexp.MustCompile(`(?i)\bauthorization\b\s*[:=]\s*(?:Bearer|Basic)\s+[^\s;,&]+`), "authorization=<redacted>"},

		// 6) Common query parameters that may carry secrets (if query not stripped for some reason)
		{regexp.MustCompile(`(?i)([?&](?:token|api[_-]?token|access[_-]?token|password|pwd|auth|authorization)=)([^&\s]+)`), `$1<redacted>`},

		// 7) Token or secret-like assignments in text (key[:=]value)
		{regexp.MustCompile(`(?i)\b(api[_-]?token|token|access[_-]?token|refresh[_-]?token|client[_-]?secret|secret|password|pwd|x[_-]?api[_-]?key|x[_-]?api[_-]?token)\b\s*:\s*([^\s;,&]+)`), `$1: <redacted>`},
		{regexp.MustCompile(`(?i)\b(api[_-]?token|token|access[_-]?token|refresh[_-]?token|client[_-]?secret|secret|password|pwd|x[_-]?api[_-]?key|x[_-]?api[_-]?token)\b\s*=\s*([^\s;,&]+)`), `$1=<redacted>`},

		// 8) JSON-style key redaction (double or single quotes)
		{regexp.MustCompile(`(?i)"(access_token|refresh_token|api[_-]?token|token|client_secret|secret|password|pwd|authorization)"\s*:\s*"[^"]*"`), `"$1":"<redacted>"`},
		{regexp.MustCompile(`(?i)'(access_token|refresh_token|api[_-]?token|token|client_secret|secret|password|pwd|authorization)'\s*:\s*'[^']*'`), `'${1}':'<redacted>'`},
	}

	out := s
	for _, p := range patterns {
		out = p.re.ReplaceAllString(out, p.repl)
	}

	// Emails: mask the local-part except first character, keep domain intact.
	// e.g., john.doe@example.com -> j***@example.com
	maskEmail := regexp.MustCompile(`([A-Za-z0-9._%+\-])[A-Za-z0-9._%+\-]*(@[A-Za-z0-9.\-]+\.[A-Za-z]{2,})`)
	out = maskEmail.ReplaceAllString(out, `${1}***${2}`)

	return out
}

// RedactHeaders returns a safe copy of the provided headers with sensitive
// keys redacted. It operates on a copy; the input is not mutated.
func RedactHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	redacted := http.Header{}
	sensitive := map[string]struct{}{
		"Authorization":       {},
		"Proxy-Authorization": {},
		"Cookie":              {},
		"Set-Cookie":          {},
		"X-Atlassian-Token":   {},
	}
	for k, vals := range h {
		ck := http.CanonicalHeaderKey(k)
		if _, ok := sensitive[ck]; ok {
			redacted[ck] = []string{"<redacted>"}
			continue
		}
		// Copy non-sensitive values
		cpy := make([]string, len(vals))
		copy(cpy, vals)
		// extra safety in case values include tokens accidentally
		for i := range cpy {
			cpy[i] = RedactSecrets(cpy[i])
		}
		redacted[ck] = cpy
	}
	return redacted
}

// RedactJoin is a small helper to join strings with a separator after redaction.
func RedactJoin(parts []string, sep string) string {
	for i := range parts {
		parts[i] = RedactSecrets(parts[i])
	}
	return strings.Join(parts, sep)
}
