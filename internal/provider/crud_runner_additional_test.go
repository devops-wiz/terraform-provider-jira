// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ensure helper wiring
func ensureTo(diags *diag.Diagnostics) func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
	return func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, diags, opts)
	}
}

func TestCRUDRunner_DoRead_getStateError_ShortCircuits(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	apiReadCalls := 0
	ensureCalls := 0

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			apiReadCalls++
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		ensureCalls++
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, &diags, opts)
	}
	d := r.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("state error", "cannot read state")
			return d
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			t.Fatalf("setState should not be called")
			return nil
		},
		func(ctx context.Context) { t.Fatalf("remove should not be called") },
		ensure,
		HTTPStatusFromScheme,
	)
	if !d.HasError() {
		t.Fatalf("expected diagnostics from getState error")
	}
	if apiReadCalls != 0 {
		t.Fatalf("expected APIRead not to be called, got %d", apiReadCalls)
	}
	if ensureCalls != 0 {
		t.Fatalf("expected ensure not to be called, got %d", ensureCalls)
	}
}

func TestCRUDRunner_DoUpdate_getPlanError_ShortCircuits(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	buildCalls := 0
	ensureCalls := 0

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			buildCalls++
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		ensureCalls++
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, &diags, opts)
	}
	d := r.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("plan error", "cannot read plan")
			return d
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			t.Fatalf("setState should not be called")
			return nil
		},
		ensure,
	)
	if !d.HasError() {
		t.Fatalf("expected diagnostics from getPlan error")
	}
	if buildCalls != 0 {
		t.Fatalf("expected BuildPayload not to be called, got %d", buildCalls)
	}
	if ensureCalls != 0 {
		t.Fatalf("expected ensure not to be called, got %d", ensureCalls)
	}
}

func TestCRUDRunner_DoDelete_getStateError_ShortCircuits(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	apiDelCalls := 0
	ensureCalls := 0

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			apiDelCalls++
			return testhelpers.MkRS(204, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		ensureCalls++
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, &diags, opts)
	}
	d := r.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("state error", "cannot read state")
			return d
		},
		ensure,
	)
	if !d.HasError() {
		t.Fatalf("expected diagnostics from getState error on delete")
	}
	if apiDelCalls != 0 {
		t.Fatalf("expected APIDelete not called, got %d", apiDelCalls)
	}
	if ensureCalls != 0 {
		t.Fatalf("expected ensure not called, got %d", ensureCalls)
	}
}

func TestCRUDRunner_Create_APICreate_NilResponse_WithError(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	mapCalls := 0

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, nil, errors.New("boom")
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			return nil
		},
	}
	r := NewCRUDRunner(h)
	_ = r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			t.Fatalf("setState should not be called")
			return nil
		},
		ensureTo(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from ensure with nil response + error")
	}
	if mapCalls != 0 {
		t.Fatalf("expected no mapping when create fails, got %d", mapCalls)
	}
}
