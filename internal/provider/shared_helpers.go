// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

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
