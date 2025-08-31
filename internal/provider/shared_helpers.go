// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Tiny mapping helpers to reduce verbosity in map-to-state code.
func stringOrNull(s string) types.String {
	if s != "" {
		return types.StringValue(s)
	}
	return types.StringNull()
}

func urlOrNull(s string) types.String { // alias for clarity in models with URLs
	return stringOrNull(s)
}

func int64OrNull(v int64) types.Int64 {
	if v != 0 {
		return types.Int64Value(v)
	}
	return types.Int64Null()
}

func boolValue(b bool) types.Bool { return types.BoolValue(b) }

// ensureWith wraps EnsureSuccessOrDiagFromSchemeWithOptions binding the diagnostics pointer.
// Use in Resource CRUD/Import methods to avoid repeating the closure at each callsite.
func ensureWith(diags *diag.Diagnostics) func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
	return func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, diags, opts)
	}
}

// withTimeout wraps ctx with a timeout when d > 0. If d <= 0, it returns the
// original context and a no-op cancel, allowing callers to `defer cancel()` unconditionally.
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	ctx2, cancel := context.WithTimeout(ctx, d)
	// Debug-level log to aid troubleshooting: indicates an operation-scoped timeout is applied.
	// Avoids sensitive data and logs only non-PII duration.
	tflog.Debug(ctx2, "context deadline set for operation", map[string]interface{}{"timeout": d.String()})
	return ctx2, cancel
}

// listHasUnknown reports if the list itself or any of its elements are unknown.
func listHasUnknown(l types.List) bool {
	if l.IsUnknown() {
		return true
	}
	elems := l.Elements()
	for i := range elems {
		if elems[i].IsUnknown() {
			return true
		}
	}
	return false
}

// getKnownStrings parses a Terraform list of strings into a Go slice.
// Returns (nil, true) if the list or any of its elements are unknown at plan time, so the caller can defer evaluation.
// On conversion failures with known values, records an attribute-scoped error and returns (nil, false).
func getKnownStrings(ctx context.Context, l types.List, attr string, diags *diag.Diagnostics) (vals []string, deferEval bool) {
	if l.IsNull() {
		return nil, false
	}
	if listHasUnknown(l) {
		return nil, true
	}
	vals = make([]string, len(l.Elements()))
	if d := l.ElementsAs(ctx, &vals, false); d.HasError() {
		diags.AddAttributeError(
			path.Root(attr),
			fmt.Sprintf("Invalid %s list", attr),
			fmt.Sprintf("Failed to read '%s' as a list of strings. Ensure all elements are known and of type string.", attr),
		)
		diags.Append(d...)
		return nil, false
	}
	return vals, false
}

// uniqueStrings returns a de-duplicated slice preserving first occurrence order.
func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
