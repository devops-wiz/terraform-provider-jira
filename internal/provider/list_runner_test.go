// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type item = *models.IssueTypeScheme

type out = workTypeResourceModel

func TestListRunner_ListWithoutFilter(t *testing.T) {
	ctx := context.Background()
	h := ListHooks[item, out]{
		List: func(ctx context.Context) ([]item, diag.Diagnostics) {
			return []item{
				&models.IssueTypeScheme{ID: "a", HierarchyLevel: 1},
				&models.IssueTypeScheme{ID: "b", HierarchyLevel: 2},
			}, nil
		},
		KeyOf: func(i item) string { return i.ID },
		MapToOut: func(ctx context.Context, i item) (out, diag.Diagnostics) {
			var m out
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := DoListToMap(ctx, h)
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

func TestListRunner_ListWithFilter(t *testing.T) {
	ctx := context.Background()
	h := ListHooks[item, out]{
		List: func(ctx context.Context) ([]item, diag.Diagnostics) {
			return []item{
				&models.IssueTypeScheme{ID: "a", HierarchyLevel: 1},
				&models.IssueTypeScheme{ID: "b", HierarchyLevel: 2},
			}, nil
		},
		Filter: func(ctx context.Context, i item) bool { return i.HierarchyLevel%2 == 0 },
		KeyOf:  func(i item) string { return i.ID },
		MapToOut: func(ctx context.Context, i item) (out, diag.Diagnostics) {
			var m out
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := DoListToMap(ctx, h)
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

func TestListRunner_MapToOutErrorPropagates(t *testing.T) {
	ctx := context.Background()
	calls := 0
	h := ListHooks[item, out]{
		List: func(ctx context.Context) ([]item, diag.Diagnostics) {
			return []item{&models.IssueTypeScheme{ID: "a"}}, nil
		},
		KeyOf: func(i item) string { return i.ID },
		MapToOut: func(ctx context.Context, i item) (out, diag.Diagnostics) {
			calls++
			var d diag.Diagnostics
			d.AddError("map error", "failed mapping")
			return out{}, d
		},
	}
	_, diags := DoListToMap(ctx, h)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics due to mapping error")
	}
	if calls != 1 {
		t.Fatalf("expected single MapToOut call, got %d", calls)
	}
}

func TestListRunner_EmptyList_WithFilter_ReturnsEmptyMap(t *testing.T) {
	ctx := context.Background()
	h := ListHooks[item, out]{
		List: func(ctx context.Context) ([]item, diag.Diagnostics) {
			return []item{}, nil
		},
		Filter: func(ctx context.Context, i item) bool { return i.HierarchyLevel%2 == 0 },
		KeyOf:  func(i item) string { return i.ID },
		MapToOut: func(ctx context.Context, i item) (out, diag.Diagnostics) {
			var m out
			d := mapWorkTypeSchemeToModel(ctx, i, &m)
			return m, d
		},
	}
	m, diags := DoListToMap(ctx, h)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map for empty list, got %d", len(m))
	}
}
