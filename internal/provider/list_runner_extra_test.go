// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strconv"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Aliases to satisfy constrained generics
type item2 = *models.IssueTypeScheme
type out2 = workTypeResourceModel

func TestListRunner_ListDiagnostics_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mapCalls := 0
	h := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddError("list failed", "service unavailable")
			// even if items are present, error should short-circuit processing
			return []item2{&models.IssueTypeScheme{ID: "a"}}, d
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			mapCalls++
			return out2{}, nil
		},
	}
	m, diags := DoListToMap(ctx, h)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics error from List")
	}
	if m != nil {
		t.Fatalf("expected nil result map on error, got %#v", m)
	}
	if mapCalls != 0 {
		t.Fatalf("expected MapToOut not to be called on list error, got %d", mapCalls)
	}
}

func TestListRunner_ListDiagnostics_Warning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal warning")
			return []item2{&models.IssueTypeScheme{ID: "a"}, &models.IssueTypeScheme{ID: "b"}}, d
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("did not expect error diagnostics, got: %v", diags)
	}
	if len(diags) == 0 {
		t.Fatalf("expected warnings from List to be preserved in diagnostics")
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 items, got %d", len(m))
	}
}

func TestListRunner_DuplicateKeys_LastWriteWins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	A := &models.IssueTypeScheme{ID: "k", Name: "first"}
	B := &models.IssueTypeScheme{ID: "k", Name: "second"}
	C := &models.IssueTypeScheme{ID: "z", Name: "other"}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return []item2{A, B, C}, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			// use shared mapper to keep semantics identical to data source behavior
			return func() (out2, diag.Diagnostics) { var mm out2; d := mapWorkTypeSchemeToModel(ctx, i, &mm); return mm, d }()
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(m))
	}
	if got := m["k"].Name.ValueString(); got != "second" {
		t.Fatalf("last write wins violated: got %q want %q", got, "second")
	}
	// Document behavior: last write wins when duplicate keys occur.
}

func TestListRunner_EmptyKey_Included(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			return []item2{&models.IssueTypeScheme{ID: "", Name: "empty"}}, nil
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			m.Name = types.StringValue(i.Name)
			return m, nil
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m))
	}
	if _, ok := m[""]; !ok {
		t.Fatalf("expected presence of empty key \"\"")
	}
}

func TestListRunner_ListNilSlice_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := ListHooks[item2, out2]{
		List:     func(ctx context.Context) ([]item2, diag.Diagnostics) { return nil, nil },
		KeyOf:    func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) { return out2{}, nil },
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m == nil {
		t.Fatalf("expected non-nil empty map")
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d", len(m))
	}
}

func TestListRunner_FilterAllFalse_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			return []item2{&models.IssueTypeScheme{ID: "a"}, &models.IssueTypeScheme{ID: "b"}}, nil
		},
		Filter:   func(ctx context.Context, i item2) bool { return false },
		KeyOf:    func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) { return out2{}, nil },
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map when filter excludes all, got %d", len(m))
	}
}

func TestListRunner_MapToOutError_MidStream_Stops_NoPartial(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	calls := 0
	n := 3
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			calls++
			if i.ID == "1" {
				var d diag.Diagnostics
				d.AddError("map fail", "bad item")
				return out2{}, d
			}
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m, diags := DoListToMap(ctx, h)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics due to mapping error")
	}
	if m != nil {
		t.Fatalf("expected nil result on mapping error")
	}
	if calls != 2 {
		t.Fatalf("expected mapping to stop at first error, got calls=%d", calls)
	}
}

func TestListRunner_MapToOutWarning_Included(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			return []item2{&models.IssueTypeScheme{ID: "a"}}, nil
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal map warning")
			return m, d
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics error: %v", diags)
	}
	if len(diags) == 0 {
		t.Fatalf("expected warning diagnostics to be returned")
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 item included, got %d", len(m))
	}
}

func TestListRunner_FilterCornerCases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Always true: include all
	h1 := ListHooks[item2, out2]{
		List: func(ctx context.Context) ([]item2, diag.Diagnostics) {
			return []item2{&models.IssueTypeScheme{ID: "a"}, &models.IssueTypeScheme{ID: "b"}, &models.IssueTypeScheme{ID: "c"}}, nil
		},
		Filter: func(ctx context.Context, i item2) bool { return true },
		KeyOf:  func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m1, d1 := DoListToMap(ctx, h1)
	if d1.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d1)
	}
	if len(m1) != 3 {
		t.Fatalf("expected 3 items, got %d", len(m1))
	}

	// Every Nth (10th)
	n := 100
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h2 := ListHooks[item2, out2]{
		List:   func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		Filter: func(ctx context.Context, i item2) bool { v, _ := strconv.Atoi(i.ID); return v%10 == 0 },
		KeyOf:  func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m2, d2 := DoListToMap(ctx, h2)
	if d2.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d2)
	}
	if len(m2) != 10 {
		t.Fatalf("expected 10 sampled items, got %d", len(m2))
	}
	for _, k := range []string{"0", "10", "90"} {
		if _, ok := m2[k]; !ok {
			t.Fatalf("expected key %q present", k)
		}
	}
}

func TestListRunner_HugeList_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping huge list stress test in short mode")
	}
	ctx := context.Background()
	n := 50000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			// minimal mapping to keep memory reasonable
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != n {
		t.Fatalf("expected %d items, got %d", n, len(m))
	}
	for _, k := range []string{"0", "12345", strconv.Itoa(n - 1)} {
		if v, ok := m[k]; !ok || v.ID.ValueString() != k {
			t.Fatalf("spot-check failed for key %q", k)
		}
	}
}

func BenchmarkDoListToMap_Huge(b *testing.B) {
	n := 10000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = DoListToMap(ctx, h)
	}
}

func TestDoListToMapWithLimit_MaxItemsCap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	ctx := context.Background()
	n := 10000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 1000, WarnThreshold: 0, PreallocCap: 0, RespectContext: false}
	m, diags := DoListToMapWithLimit[item2, out2](ctx, h, opts)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 1000 {
		t.Fatalf("expected 1000 items due to cap, got %d", len(m))
	}
	if len(diags) == 0 {
		t.Fatalf("expected a warning about result capping")
	}
}

func TestDoListToMapWithLimit_WarnThresholdOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	ctx := context.Background()
	n := 5000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 0, WarnThreshold: 1000, PreallocCap: 0, RespectContext: false}
	m, diags := DoListToMapWithLimit[item2, out2](ctx, h, opts)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != n {
		t.Fatalf("expected full result %d, got %d", n, len(m))
	}
	if len(diags) == 0 {
		t.Fatalf("expected a warning about large result set")
	}
}

func TestDoListToMapWithLimit_RespectContext_CancelEarly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := 5000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[item2, out2]{
		List:  func(ctx context.Context) ([]item2, diag.Diagnostics) { return items, nil },
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{RespectContext: true}
	// Cancel immediately; loop checks every 1000 items, so expect partial < n
	cancel()
	m, diags := DoListToMapWithLimit[item2, out2](ctx, h, opts)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if len(diags) == 0 {
		t.Fatalf("expected a cancellation warning")
	}
	if len(m) == 0 || len(m) >= n {
		t.Fatalf("expected partial results due to cancellation; got %d of %d", len(m), n)
	}
}

func TestDoListToMapWithLimit_Paginated_Aggregates_And_RespectsMax(t *testing.T) {
	ctx := context.Background()
	n := 23
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	page := func(startAt, max int) ([]item2, bool) {
		if startAt >= len(items) {
			return []item2{}, true
		}
		end := startAt + max
		if end > len(items) {
			end = len(items)
		}
		return items[startAt:end], end == len(items)
	}
	h := ListHooks[item2, out2]{
		ListPage: func(ctx context.Context, startAt, max int) ([]item2, bool, diag.Diagnostics) {
			its, last := page(startAt, max)
			return its, last, nil
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 12}
	m, diags := DoListToMapWithLimit[item2, out2](ctx, h, opts)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 12 {
		t.Fatalf("expected 12 items due to cap, got %d", len(m))
	}
	if len(diags) == 0 {
		t.Fatalf("expected a capping warning in diagnostics")
	}
}

func TestDoListToMapWithLimit_Paginated_CancellationMidStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := 5000
	items := make([]item2, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	page := func(startAt, max int) ([]item2, bool) {
		if startAt >= len(items) {
			return []item2{}, true
		}
		end := startAt + max
		if end > len(items) {
			end = len(items)
		}
		// trigger cancel after first page delivered
		if startAt == 0 {
			cancel()
		}
		return items[startAt:end], end == len(items)
	}
	h := ListHooks[item2, out2]{
		ListPage: func(ctx context.Context, startAt, max int) ([]item2, bool, diag.Diagnostics) {
			its, last := page(startAt, max)
			return its, last, nil
		},
		KeyOf: func(i item2) string { return i.ID },
		MapToOut: func(ctx context.Context, i item2) (out2, diag.Diagnostics) {
			var m out2
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{RespectContext: true}
	m, diags := DoListToMapWithLimit[item2, out2](ctx, h, opts)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if len(diags) == 0 {
		t.Fatalf("expected a cancellation warning in diagnostics")
	}
	if len(m) == 0 || len(m) >= n {
		t.Fatalf("expected partial results due to cancellation; got %d of %d", len(m), n)
	}
}
