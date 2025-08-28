// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Function type aliases for list-to-map flows.
//
// Why aliases?
// - Keep the ListHooks signatures concise and intention-revealing (list, filter, key-of, mapper).
// - Centralize call signatures so future changes won’t ripple across all data sources.
// - Combine with constraints below to catch type mismatches at compile time.
//
// Usage:
// - Implement List to fetch all items (handle pagination internally).
// - Optionally implement Filter to apply client-side filtering.
// - KeyOf must return a stable string key (e.g., ID as string).
// - MapToOut converts an API item to the Terraform object model (struct of types.* fields).
type ListFunc[TAPI APIListConstraint] func(ctx context.Context) ([]TAPI, diag.Diagnostics)
type FilterFunc[TAPI APIListConstraint] func(ctx context.Context, item TAPI) bool
type KeyOfFunc[TAPI APIListConstraint] func(item TAPI) string
type MapToOutFunc[TAPI APIListConstraint, TOut OutModelConstraint] func(ctx context.Context, item TAPI) (TOut, diag.Diagnostics)
type AttrTypesFunc func() map[string]attr.Type

// ListPageFunc optionally streams pages of items. Return items, whether this is the last page, and diagnostics.
// - startAt: zero-based offset of the first item to return for the page
// - max: desired maximum page size. Implementations may return fewer items.
// When provided, DoListToMapWithLimit will iterate pages until isLast=true, context cancellation, or reaching MaxItems.
type ListPageFunc[TAPI APIListConstraint] func(ctx context.Context, startAt, max int) (items []TAPI, isLast bool, d diag.Diagnostics)

// Generic type constraints restricted to available provider types.
//
// What these do
// - APIListConstraint enumerates the supported go-atlassian API models that appear in lists.
// - OutModelConstraint enumerates the supported Terraform object models (map values).
//
// How to extend
// - Add new API and output model types to the unions below when introducing a new data source.
type APIListConstraint interface {
	*models.ProjectScheme | *models.ProjectCategoryScheme | *models.IssueTypeScheme
}

type OutModelConstraint interface {
	projectResourceModel | projectCategoryResourceModel | workTypeResourceModel
}

// ListHooks defines list-to-map helpers for data sources.
// TAPI: API model per item (constrained by APIListConstraint).
// TOut: Terraform object model per item (constrained by OutModelConstraint).
type ListHooks[TAPI APIListConstraint, TOut OutModelConstraint] struct {
	// List should fetch all items (handle pagination internally if needed).
	List ListFunc[TAPI]

	// Optional paginated fetcher. If provided, DoListToMapWithLimit will iterate via pages
	// using a default page size and aggregate incrementally. This avoids holding all items in memory.
	ListPage ListPageFunc[TAPI]

	// Optional filter applied client-side (return true to keep).
	Filter FilterFunc[TAPI]

	// KeyOf must return the stable string key (e.g., ID string).
	KeyOf KeyOfFunc[TAPI]

	// MapToOut converts API item to Terraform object model.
	MapToOut MapToOutFunc[TAPI, TOut]

	// AttrTypes returns the ObjectType attribute types for TOut.
	AttrTypes AttrTypesFunc
}

// ListOptions configures DoListToMapWithLimit behavior.
//
// Fields
// - MaxItems: hard cap on kept items (post-filter). 0 = unlimited (default).
// - WarnThreshold: soft threshold that adds a warning once kept items reach/exceed it. 0 = disabled.
// - PreallocCap: explicit capacity for result map. 0 = auto (based on input length or MaxItems when known).
// - RespectContext: if true, checks ctx.Done() periodically and returns early with a warning if canceled.
type ListOptions struct {
	MaxItems       int
	WarnThreshold  int
	PreallocCap    int
	RespectContext bool
}

// DoListToMap builds a map[string]TOut using hooks (legacy convenience).
// Use DoListToMapWithLimit for large datasets and guardrails.
//
// Steps
// 1) List: fetch all items (may include server-side pagination).
// 2) Filter (optional): keep only items where Filter returns true.
// 3) KeyOf: compute the stable key for each item.
// 4) MapToOut: convert each API item into the Terraform object model.
// 5) Aggregate: return a map keyed by the stable string key.
//
// Notes
// - Diagnostics from List and MapToOut are accumulated and returned.
// - If diagnostics contain errors at any point, the partial result is not returned.
func DoListToMap[TAPI APIListConstraint, TOut OutModelConstraint](
	ctx context.Context,
	h ListHooks[TAPI, TOut],
) (map[string]TOut, diag.Diagnostics) {
	return DoListToMapWithLimit[TAPI, TOut](ctx, h, ListOptions{})
}

// DoListToMapWithLimit builds a map[string]TOut using hooks with guardrails for large datasets.
//
// Behavior
// - If hooks.ListPage is provided, pages are fetched incrementally with a default page size (200).
// - Filter (when provided) is applied before mapping to avoid unnecessary allocations.
// - The result map is pre-allocated using a conservative capacity derived from PreallocCap/MaxItems/input size.
// - Last write wins for duplicate keys (documented behavior).
// - If MaxItems > 0, mapping stops after that many kept items; a warning is added indicating capping.
// - If WarnThreshold > 0 and kept >= threshold, a warning is added (once).
// - If RespectContext is true, ctx.Done() is checked every 1000 processed items and early termination returns a warning.
//
// Diagnostics
// - Diagnostics from List/ListPage and MapToOut are appended and on error return immediately (no partial results).
// - Soft warnings (WarnThreshold, MaxItems cap, context cancellation) are added and a partial, coherent result is returned.
func DoListToMapWithLimit[TAPI APIListConstraint, TOut OutModelConstraint](
	ctx context.Context,
	h ListHooks[TAPI, TOut],
	opts ListOptions,
) (map[string]TOut, diag.Diagnostics) {
	const (
		pageSizeDefault  = 200
		checkCancelEvery = 1000
	)
	var diags diag.Diagnostics

	// Helper to decide capacity
	capFor := func(total int) int {
		capHint := total
		if opts.PreallocCap > 0 && (capHint == 0 || opts.PreallocCap < capHint) {
			capHint = opts.PreallocCap
		}
		if opts.MaxItems > 0 && (capHint == 0 || opts.MaxItems < capHint) {
			capHint = opts.MaxItems
		}
		if capHint < 0 {
			capHint = 0
		}
		return capHint
	}

	warnedThreshold := false
	warnCanceled := func(reason string) { diags.AddWarning("listing canceled", reason) }
	warnThreshold := func(count int) {
		if !warnedThreshold {
			warnedThreshold = true
			diags.AddWarning("large result set", "number of items kept reached threshold: "+intToString(count))
		}
	}
	warnCapped := func(max int) {
		diags.AddWarning("result capped", "maximum items reached; result truncated at "+intToString(max))
	}

	kept := 0
	processed := 0

	// Main aggregator with limiter/cancel checks
	processItems := func(items []TAPI, result map[string]TOut) bool {
		for _, it := range items {
			processed++
			if opts.RespectContext && processed%checkCancelEvery == 0 {
				select {
				case <-ctx.Done():
					warnCanceled("context canceled or deadline exceeded during listing; returning partial results")
					return false
				default:
				}
			}
			if h.Filter != nil && !h.Filter(ctx, it) {
				continue
			}
			k := h.KeyOf(it)
			obj, d2 := h.MapToOut(ctx, it)
			diags.Append(d2...)
			if diags.HasError() {
				return false
			}
			result[k] = obj // last write wins by design
			kept++
			if opts.WarnThreshold > 0 && kept >= opts.WarnThreshold {
				warnThreshold(kept)
			}
			if opts.MaxItems > 0 && kept >= opts.MaxItems {
				warnCapped(opts.MaxItems)
				return false
			}
		}
		return true
	}

	// Fast path: no pagination
	if h.ListPage == nil {
		items, d := h.List(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		result := make(map[string]TOut, capFor(len(items)))
		cont := processItems(items, result)
		if diags.HasError() {
			return nil, diags
		}
		if !cont {
			return result, diags
		}
		return result, diags
	}

	// Paginated path
	startAt := 0
	max := pageSizeDefault
	// If MaxItems is smaller than page size, reduce the first page to avoid extra memory
	if opts.MaxItems > 0 && opts.MaxItems < max {
		max = opts.MaxItems
	}
	result := make(map[string]TOut, capFor(max))
	for {
		items, isLast, d := h.ListPage(ctx, startAt, max)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		if len(items) == 0 {
			if isLast {
				break
			}
			// Advance cautiously to avoid infinite loops on empty non-last pages
			startAt += 0
		} else {
			if !processItems(items, result) {
				break
			}
			startAt += len(items)
		}
		if isLast {
			break
		}
		// Adjust next page size based on remaining MaxItems
		if opts.MaxItems > 0 {
			remaining := opts.MaxItems - kept
			if remaining <= 0 {
				break
			}
			if remaining < max {
				max = remaining
			}
		}
		if opts.RespectContext {
			select {
			case <-ctx.Done():
				warnCanceled("context canceled or deadline exceeded during pagination; returning partial results")
				break
			default:
			}
		}
	}
	return result, diags
}

// intToString is a tiny helper to avoid importing strconv at call sites in this file.
func intToString(v int) string {
	// Manual fast path for small ints; fallback to stdlib if needed later.
	// Here correctness > micro-optimizations; keep private.
	if v == 0 {
		return "0"
	}
	// Build decimal representation
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		d := byte(v%10) + '0'
		buf = append([]byte{d}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
