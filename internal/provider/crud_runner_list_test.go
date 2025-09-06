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

// Aliases to satisfy constrained generics for Issue Types list tests
// API list item type and output model type
type listItem = *models.IssueTypeScheme
type listOut = workTypeResourceModel

func TestCRUDRunner_List_NoFilter(t *testing.T) {
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]

	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{
				&models.IssueTypeScheme{ID: "a", HierarchyLevel: 1},
				&models.IssueTypeScheme{ID: "b", HierarchyLevel: 2},
			}, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 items, got %d", len(m))
	}
	if _, ok := m["a"]; !ok {
		t.Fatalf("missing key 'a'")
	}
	if _, ok := m["b"]; !ok {
		t.Fatalf("missing key 'b'")
	}
}

func TestCRUDRunner_List_WithFilter(t *testing.T) {
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]

	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{
				&models.IssueTypeScheme{ID: "a", HierarchyLevel: 1},
				&models.IssueTypeScheme{ID: "b", HierarchyLevel: 2},
			}, nil
		},
		Filter: func(ctx context.Context, i listItem) bool { return i.HierarchyLevel%2 == 0 },
		KeyOf:  func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 item after filter, got %d", len(m))
	}
	if _, ok := m["b"]; !ok {
		t.Fatalf("expected only key 'b' to remain")
	}
}

func TestCRUDRunner_List_MapToOutError_Propagates(t *testing.T) {
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	calls := 0
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{&models.IssueTypeScheme{ID: "a"}}, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			calls++
			var d diag.Diagnostics
			d.AddError("map error", "failed mapping")
			return listOut{}, d
		},
	}
	_, diags := runner.DoListIssueTypes(ctx, h)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics due to mapping error")
	}
	if calls != 1 {
		t.Fatalf("expected single MapToOut call, got %d", calls)
	}
}

func TestCRUDRunner_List_EmptyList_WithFilter_ReturnsEmptyMap(t *testing.T) {
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List:   func(ctx context.Context) ([]listItem, diag.Diagnostics) { return []listItem{}, nil },
		Filter: func(ctx context.Context, i listItem) bool { return i.HierarchyLevel%2 == 0 },
		KeyOf:  func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map for empty list, got %d", len(m))
	}
}

func TestCRUDRunner_List_ListDiagnostics_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	mapCalls := 0
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddError("list failed", "service unavailable")
			return []listItem{&models.IssueTypeScheme{ID: "a"}}, d
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			mapCalls++
			return listOut{}, nil
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_ListDiagnostics_Warning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal warning")
			return []listItem{&models.IssueTypeScheme{ID: "a"}, &models.IssueTypeScheme{ID: "b"}}, d
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_DuplicateKeys_LastWriteWins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	A := &models.IssueTypeScheme{ID: "k", Name: "first"}
	B := &models.IssueTypeScheme{ID: "k", Name: "second"}
	C := &models.IssueTypeScheme{ID: "z", Name: "other"}
	h := ListHooks[listItem, listOut]{
		List:  func(ctx context.Context) ([]listItem, diag.Diagnostics) { return []listItem{A, B, C}, nil },
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var mm listOut
			d := mapWorkTypeSchemeToModel(ctx, i, &mm)
			return mm, d
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(m))
	}
	if got := m["k"].Name.ValueString(); got != "second" {
		t.Fatalf("last write wins violated: got %q want %q", got, "second")
	}
}

func TestCRUDRunner_List_EmptyKey_Included(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{&models.IssueTypeScheme{ID: "", Name: "empty"}}, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			m.Name = types.StringValue(i.Name)
			return m, nil
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_ListNilSlice_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List:     func(ctx context.Context) ([]listItem, diag.Diagnostics) { return nil, nil },
		KeyOf:    func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) { return listOut{}, nil },
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_FilterAllFalse_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{&models.IssueTypeScheme{ID: "a"}, &models.IssueTypeScheme{ID: "b"}}, nil
		},
		Filter:   func(ctx context.Context, i listItem) bool { return false },
		KeyOf:    func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) { return listOut{}, nil },
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map when filter excludes all, got %d", len(m))
	}
}

func TestCRUDRunner_List_MapToOutError_MidStream_Stops_NoPartial(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	calls := 0
	n := 3
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[listItem, listOut]{
		List:  func(ctx context.Context) ([]listItem, diag.Diagnostics) { return items, nil },
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			calls++
			if i.ID == "1" {
				var d diag.Diagnostics
				d.AddError("map fail", "bad item")
				return listOut{}, d
			}
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_MapToOutWarning_Included(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	h := ListHooks[listItem, listOut]{
		List: func(ctx context.Context) ([]listItem, diag.Diagnostics) {
			return []listItem{&models.IssueTypeScheme{ID: "a"}}, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal map warning")
			return m, d
		},
	}
	m, diags := runner.DoListIssueTypes(ctx, h)
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

func TestCRUDRunner_List_WithLimit_MaxItemsCap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	n := 10000
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[listItem, listOut]{
		List:  func(ctx context.Context) ([]listItem, diag.Diagnostics) { return items, nil },
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 1000, WarnThreshold: 0, PreallocCap: 0, RespectContext: false}
	m, diags := runner.DoListIssueTypesWithLimit(ctx, h, opts)
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

func TestCRUDRunner_List_WithLimit_WarnThresholdOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	n := 5000
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[listItem, listOut]{
		List:  func(ctx context.Context) ([]listItem, diag.Diagnostics) { return items, nil },
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 0, WarnThreshold: 1000, PreallocCap: 0, RespectContext: false}
	m, diags := runner.DoListIssueTypesWithLimit(ctx, h, opts)
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

func TestCRUDRunner_List_WithLimit_RespectContext_CancelEarly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	n := 5000
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	h := ListHooks[listItem, listOut]{
		List:  func(ctx context.Context) ([]listItem, diag.Diagnostics) { return items, nil },
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{RespectContext: true}
	cancel()
	m, diags := runner.DoListIssueTypesWithLimit(ctx, h, opts)
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

func TestCRUDRunner_List_WithLimit_Paginated_Aggregates_And_RespectsMax(t *testing.T) {
	ctx := context.Background()
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	n := 23
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	page := func(startAt, max int) ([]listItem, bool) {
		if startAt >= len(items) {
			return []listItem{}, true
		}
		end := startAt + max
		if end > len(items) {
			end = len(items)
		}
		return items[startAt:end], end == len(items)
	}
	h := ListHooks[listItem, listOut]{
		ListPage: func(ctx context.Context, startAt, max int) ([]listItem, bool, diag.Diagnostics) {
			its, last := page(startAt, max)
			return its, last, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{MaxItems: 12}
	m, diags := runner.DoListIssueTypesWithLimit(ctx, h, opts)
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

func TestCRUDRunner_List_WithLimit_Paginated_CancellationMidStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var runner CRUDRunner[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]
	n := 5000
	items := make([]listItem, n)
	for i := 0; i < n; i++ {
		items[i] = &models.IssueTypeScheme{ID: strconv.Itoa(i)}
	}
	page := func(startAt, max int) ([]listItem, bool) {
		if startAt >= len(items) {
			return []listItem{}, true
		}
		end := startAt + max
		if end > len(items) {
			end = len(items)
		}
		if startAt == 0 {
			cancel()
		}
		return items[startAt:end], end == len(items)
	}
	h := ListHooks[listItem, listOut]{
		ListPage: func(ctx context.Context, startAt, max int) ([]listItem, bool, diag.Diagnostics) {
			its, last := page(startAt, max)
			return its, last, nil
		},
		KeyOf: func(i listItem) string { return i.ID },
		MapToOut: func(ctx context.Context, i listItem) (listOut, diag.Diagnostics) {
			var m listOut
			m.ID = types.StringValue(i.ID)
			return m, nil
		},
	}
	opts := ListOptions{RespectContext: true}
	m, diags := runner.DoListIssueTypesWithLimit(ctx, h, opts)
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
