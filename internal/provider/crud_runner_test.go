// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/devops-wiz/terraform-provider-jira/internal/provider/testhelpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ensure helper wiring: use real EnsureSuccessOrDiagFromSchemeWithOptions into local diags
func makeEnsure(diags *diag.Diagnostics) func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
	return func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, diags, opts)
	}
}

func TestCRUDRunner_Create_Update_Delete_Read_HappyPaths(t *testing.T) {
	ctx := context.Background()

	var mapCalls int
	var setCalls int
	var diags diag.Diagnostics
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, diag.Diagnostics{}
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "created"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "read"}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "updated"}, testhelpers.MkRS(204, nil, ""), nil
		},
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(204, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			st.ID = types.StringValue("id")
			return diag.Diagnostics{}
		},
		TreatDelete404AsSuccess: true,
	}
	r := NewCRUDRunner(h)

	// Create
	cDiags := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { setCalls++; return nil },
		makeEnsure(&diags),
	)
	if cDiags.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on create: %v %v", cDiags, diags)
	}
	if mapCalls != 1 {
		t.Fatalf("expected MapToState once on create, got %d", mapCalls)
	}

	// Read
	mapCalls = 0
	rdDiags := r.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { setCalls++; return nil },
		func(ctx context.Context) {},
		makeEnsure(&diags),
		HTTPStatusFromScheme,
	)
	if rdDiags.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on read: %v %v", rdDiags, diags)
	}
	if mapCalls != 1 {
		t.Fatalf("expected MapToState once on read, got %d", mapCalls)
	}

	// Update (204 acceptable)
	mapCalls = 0
	upDiags := r.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { setCalls++; return nil },
		makeEnsure(&diags),
	)
	if upDiags.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on update: %v %v", upDiags, diags)
	}
	if mapCalls != 1 {
		t.Fatalf("expected MapToState once on update, got %d", mapCalls)
	}

	// Delete (204 acceptable)
	diags = diag.Diagnostics{}
	dlDiags := r.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		makeEnsure(&diags),
	)
	if dlDiags.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on delete: %v %v", dlDiags, diags)
	}
	_ = setCalls
}

func TestCRUDRunner_Delete_404_TreatedAsSuccess(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{}, nil
		},
		APICreate: nil,
		APIRead:   nil,
		APIUpdate: nil,
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(404, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
		TreatDelete404AsSuccess: true,
	}
	r := NewCRUDRunner(h)
	d := r.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		makeEnsure(&diags),
	)
	if d.HasError() || diags.HasError() {
		t.Fatalf("expected 404 delete to be treated as success, got diags: %v %v", d, diags)
	}
}

func TestCRUDRunner_MapToStateError_Propagates(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var mapCalls int
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "created"}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "read"}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "updated"}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(204, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			var d diag.Diagnostics
			d.AddError("map error", "failed mapping")
			return d
		},
	}
	r := NewCRUDRunner(h)
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !cd.HasError() {
		t.Fatalf("expected mapping error to propagate in diagnostics")
	}
	if mapCalls != 1 {
		t.Fatalf("expected MapToState once, got %d", mapCalls)
	}
}

func TestCRUDRunner_Read_404_RemovesState_NoError(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var ensureCalls int
	var mapCalls int
	removed := false

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{}, nil
		},
		APICreate: nil,
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(404, nil, ""), nil
		},
		APIUpdate: nil,
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
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
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context) { removed = true },
		ensure,
		HTTPStatusFromScheme,
	)
	if d.HasError() || diags.HasError() {
		t.Fatalf("expected no diagnostics on 404 read removal, got: %v %v", d, diags)
	}
	if !removed {
		t.Fatalf("expected remove() to be called on 404")
	}
	if ensureCalls != 0 {
		t.Fatalf("expected ensure not to be called on 404 path, got %d", ensureCalls)
	}
	if mapCalls != 0 {
		t.Fatalf("expected MapToState not to be called on 404 path, got %d", mapCalls)
	}
}

func TestCRUDRunner_PostCreateRead_CalledAndMapped(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var mapCalls int
	var ensureCalls int
	postReadCalled := false
	var finalID string

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "created"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead:   nil,
		APIUpdate: nil,
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			st.ID = types.StringValue(api.ID)
			return nil
		},
		PostCreate: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			postReadCalled = true
			return &models.IssueTypeScheme{ID: "post"}, testhelpers.MkRS(200, nil, ""), nil
		},
	}
	r := NewCRUDRunner(h)
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		ensureCalls++
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, &diags, opts)
	}
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			finalID = src.ID.ValueString()
			return nil
		},
		ensure,
	)
	if cd.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on post-create-read: %v %v", cd, diags)
	}
	if !postReadCalled {
		t.Fatalf("expected PostCreate to be called")
	}
	if ensureCalls != 2 {
		t.Fatalf("expected ensure to be called twice (create & post-read), got %d", ensureCalls)
	}
	if mapCalls != 1 {
		t.Fatalf("expected single MapToState call, got %d", mapCalls)
	}
	if finalID != "post" {
		t.Fatalf("expected final state ID to be 'post', got %q", finalID)
	}
}

func TestCRUDRunner_StatusOverrides_201_204_Success(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var mapCalls int

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "created"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(204, nil, ""), nil
		},
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(204, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			st.ID = types.StringValue(api.ID)
			return nil
		},
		AcceptableCreateStatuses: []int{201},
		AcceptableUpdateStatuses: []int{204},
	}
	r := NewCRUDRunner(h)

	// Create
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if cd.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on create with override: %v %v", cd, diags)
	}

	// Update
	mapCalls = 0
	ud := r.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if ud.HasError() || diags.HasError() {
		t.Fatalf("unexpected diagnostics on update with override: %v %v", ud, diags)
	}
	if mapCalls != 1 {
		t.Fatalf("expected MapToState once on update, got %d", mapCalls)
	}
}

func TestCRUDRunner_EnsureFailure_AppendsDiagnostics(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(500, nil, "boom"), nil
		},
		APIRead:   nil,
		APIUpdate: nil,
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	_ = r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics to be appended by ensure on failure")
	}
	// Note: Runner may return no errors while ensure appends to external diags; that's acceptable here.
}

func TestCRUDRunner_BuildPayloadError_Propagates_NoAPICall(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var createCalls int

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddError("build error", "payload invalid")
			return nil, d
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			createCalls++
			return &models.IssueTypeScheme{ID: "x"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead:   nil,
		APIUpdate: nil,
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !cd.HasError() {
		t.Fatalf("expected diagnostics from BuildPayload to be returned")
	}
	if createCalls != 0 {
		t.Fatalf("expected APICreate not to be called on build error, got %d", createCalls)
	}
}

func TestCRUDRunner_BuildPayloadWarnings_Create_Proceeds(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var createCalls int

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal warning during build")
			return &models.IssueTypePayloadScheme{Name: "n"}, d
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			createCalls++
			return &models.IssueTypeScheme{ID: "id"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead:   nil,
		APIUpdate: nil,
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			st.ID = types.StringValue(api.ID)
			return nil
		},
	}
	r := NewCRUDRunner(h)
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if cd.HasError() || diags.HasError() {
		t.Fatalf("expected no errors, got: cd=%v diags=%v", cd, diags)
	}
	if createCalls != 1 {
		t.Fatalf("expected APICreate to be called once, got %d", createCalls)
	}
	if len(cd) == 0 {
		t.Fatalf("expected warnings from BuildPayload to be preserved in returned diagnostics")
	}
}

func TestCRUDRunner_BuildPayloadWarnings_Update_Proceeds(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var updateCalls int

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			var d diag.Diagnostics
			d.AddWarning("warn", "non-fatal warning during build")
			return &models.IssueTypePayloadScheme{Name: "n"}, d
		},
		APICreate: nil,
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			updateCalls++
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIDelete: nil,
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			st.ID = types.StringValue(api.ID)
			return nil
		},
	}
	r := NewCRUDRunner(h)
	ud := r.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if ud.HasError() || diags.HasError() {
		t.Fatalf("expected no errors, got: ud=%v diags=%v", ud, diags)
	}
	if updateCalls != 1 {
		t.Fatalf("expected APIUpdate to be called once, got %d", updateCalls)
	}
	if len(ud) == 0 {
		t.Fatalf("expected warnings from BuildPayload to be preserved in returned diagnostics")
	}
}

func TestCRUDRunner_EnsureAndAPIErrorHandling_AllOps(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create: APICreate err + 500 → ensure false; no map
	mapCalls := 0
	hCreate := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(500, nil, "boom"), errors.New("create failed")
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			return nil
		},
	}
	rc := NewCRUDRunner(hCreate)
	_ = rc.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from ensure on create error")
	}
	if mapCalls != 0 {
		t.Fatalf("expected no mapping on failed create, got %d", mapCalls)
	}

	// Read: APIRead network error
	diags = diag.Diagnostics{}
	setCalls := 0
	hRead := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(500, nil, "oops"), errors.New("net err")
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			setCalls++
			return nil
		},
	}
	rr := NewCRUDRunner(hRead)
	_ = rr.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { setCalls++; return nil },
		func(ctx context.Context) {},
		makeEnsure(&diags),
		HTTPStatusFromScheme,
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from ensure on read error")
	}
	if setCalls != 0 {
		t.Fatalf("expected no setState on failed read, got %d", setCalls)
	}

	// Update: 500
	diags = diag.Diagnostics{}
	mapCalls = 0
	hUpdate := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(500, nil, "boom"), errors.New("update failed")
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			return nil
		},
	}
	ru := NewCRUDRunner(hUpdate)
	_ = ru.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from ensure on update error")
	}
	if mapCalls != 0 {
		t.Fatalf("expected no mapping on failed update, got %d", mapCalls)
	}

	// Delete: err + 500
	diags = diag.Diagnostics{}
	hDelete := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(500, nil, "err"), errors.New("delete failed")
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	rd := NewCRUDRunner(hDelete)
	_ = rd.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from ensure on delete error")
	}
}

func TestCRUDRunner_Delete_Accepts202_WithOverride(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(202, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
		AcceptableDeleteStatuses: []int{202},
	}
	r := NewCRUDRunner(h)
	d := r.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		makeEnsure(&diags),
	)
	if d.HasError() || diags.HasError() {
		t.Fatalf("expected delete 202 to be accepted")
	}
}

func TestCRUDRunner_StatusOverrideMismatch_Fails(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "x"}, testhelpers.MkRS(500, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
		AcceptableCreateStatuses: []int{201},
	}
	r := NewCRUDRunner(h)
	_ = r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected ensure to fail when status not in overrides")
	}
}

func TestCRUDRunner_Read_403_Forbidden_NoRemove(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	removed := false

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(403, nil, "forbidden"), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	r := NewCRUDRunner(h)
	_ = r.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context) { removed = true },
		makeEnsure(&diags),
		HTTPStatusFromScheme,
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics on 403 read")
	}
	if removed {
		t.Fatalf("did not expect remove() on 403")
	}
}

func TestCRUDRunner_PostCreateRead_Error_NoMapSet(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var mapCalls, setCalls, ensureCalls int

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "x"}, testhelpers.MkRS(201, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			mapCalls++
			return nil
		},
		PostCreate: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return nil, testhelpers.MkRS(500, nil, "post-read err"), errors.New("post-read failed")
		},
	}
	r := NewCRUDRunner(h)
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		ensureCalls++
		return EnsureSuccessOrDiagFromSchemeWithOptions(ctx, action, resp, err, &diags, opts)
	}
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { setCalls++; return nil },
		ensure,
	)
	if cd.HasError() {
		t.Fatalf("runner returns no internal errors; ensure should capture diagnostics")
	}
	if mapCalls != 0 || setCalls != 0 {
		t.Fatalf("expected no map/set on post-create-read failure; map=%d set=%d", mapCalls, setCalls)
	}
	if ensureCalls != 2 {
		t.Fatalf("expected ensure twice (create and post-read), got %d", ensureCalls)
	}
	if !diags.HasError() {
		t.Fatalf("expected diagnostics from post-create-read failure")
	}
}

func TestCRUDRunner_PostCreateRead_Maps_InvalidModel_ErrorPropagates(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "x"}, testhelpers.MkRS(201, nil, ""), nil
		},
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("map error", "invalid api model")
			return d
		},
		PostCreate: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{}, testhelpers.MkRS(200, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
	}
	r := NewCRUDRunner(h)
	cd := r.DoCreate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !cd.HasError() {
		t.Fatalf("expected mapping error from post-create-read path to propagate")
	}
}

func TestCRUDRunner_ExtractID_Empty_Read_Delete(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Read with empty id
	hRead := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			if id != "" {
				t.Fatalf("expected empty id, got %q", id)
			}
			return nil, testhelpers.MkRS(400, nil, "bad id"), errors.New("bad id")
		},
		ExtractID: func(st *workTypeResourceModel) string { return "" },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	rr := NewCRUDRunner(hRead)
	_ = rr.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		func(ctx context.Context) {},
		makeEnsure(&diags),
		HTTPStatusFromScheme,
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics for empty id read")
	}

	// Delete with empty id
	diags = diag.Diagnostics{}
	hDel := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			if id != "" {
				t.Fatalf("expected empty id for delete, got %q", id)
			}
			return testhelpers.MkRS(400, nil, "bad id"), errors.New("bad id")
		},
		ExtractID: func(st *workTypeResourceModel) string { return "" },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			return nil
		},
	}
	rd := NewCRUDRunner(hDel)
	_ = rd.DoDelete(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil },
		makeEnsure(&diags),
	)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics for empty id delete")
	}
}

func TestCRUDRunner_Read_MapError_And_Update_SetStateError(t *testing.T) {
	ctx := context.Background()

	// Read MapToState error
	hRead := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("map err", "bad value")
			return d
		},
	}
	rr := NewCRUDRunner(hRead)
	d := rr.DoRead(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			t.Fatalf("setState should not be called on map error")
			return nil
		},
		func(ctx context.Context) {},
		makeEnsure(&diag.Diagnostics{}),
		HTTPStatusFromScheme,
	)
	if !d.HasError() {
		t.Fatalf("expected diagnostics from MapToState error on read")
	}

	// Update setState error
	var diags diag.Diagnostics
	hUpdate := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			st.ID = types.StringValue(api.ID)
			return nil
		},
	}
	ru := NewCRUDRunner(hUpdate)
	ud := ru.DoUpdate(ctx,
		func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
			dst.ID = types.StringValue("id")
			return nil
		},
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics {
			var d diag.Diagnostics
			d.AddError("set error", "failed to set")
			return d
		},
		makeEnsure(&diags),
	)
	if !ud.HasError() {
		t.Fatalf("expected diagnostics from setState error on update")
	}
}

func TestCRUDRunner_EnsureActionStrings_AreCorrect(t *testing.T) {
	ctx := context.Background()
	actions := []string{}
	ensure := func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool {
		actions = append(actions, action)
		// Always succeed to allow flow
		return true
	}

	// Hooks to succeed and trigger post-create read
	h := CRUDHooks[workTypeResourceModel, *models.IssueTypePayloadScheme, *models.IssueTypeScheme]{
		BuildPayload: func(ctx context.Context, st *workTypeResourceModel) (*models.IssueTypePayloadScheme, diag.Diagnostics) {
			return &models.IssueTypePayloadScheme{Name: "n"}, nil
		},
		APICreate: func(ctx context.Context, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "id"}, testhelpers.MkRS(201, nil, ""), nil
		},
		APIRead: func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIUpdate: func(ctx context.Context, id string, p *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: id}, testhelpers.MkRS(200, nil, ""), nil
		},
		APIDelete: func(ctx context.Context, id string) (*models.ResponseScheme, error) {
			return testhelpers.MkRS(204, nil, ""), nil
		},
		ExtractID: func(st *workTypeResourceModel) string { return st.ID.ValueString() },
		MapToState: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) diag.Diagnostics {
			st.ID = types.StringValue(api.ID)
			return nil
		},
		PostCreate: func(ctx context.Context, api *models.IssueTypeScheme, st *workTypeResourceModel) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
			return &models.IssueTypeScheme{ID: "full"}, testhelpers.MkRS(200, nil, ""), nil
		},
	}
	r := NewCRUDRunner(h)

	_ = r.DoCreate(ctx, func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics { return nil }, func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil }, ensure)
	_ = r.DoRead(ctx, func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
		dst.ID = types.StringValue("id")
		return nil
	}, func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil }, func(ctx context.Context) {}, ensure, HTTPStatusFromScheme)
	_ = r.DoUpdate(ctx, func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
		dst.ID = types.StringValue("id")
		return nil
	}, func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil }, ensure)
	_ = r.DoDelete(ctx, func(ctx context.Context, dst *workTypeResourceModel) diag.Diagnostics {
		dst.ID = types.StringValue("id")
		return nil
	}, ensure)

	// DoImport with an always-OK ensure
	_ = r.DoImport(
		ctx,
		"id",
		func(ctx context.Context, src *workTypeResourceModel) diag.Diagnostics { return nil },
		ensure,
	)

	expected := []string{"create resource", "post-create hook", "read resource", "update resource", "delete resource", "read imported resource"}
	if len(actions) != len(expected) {
		t.Fatalf("unexpected number of actions: got %d want %d; actions=%v", len(actions), len(expected), actions)
	}
	for i := range expected {
		if actions[i] != expected[i] {
			t.Fatalf("unexpected action at %d: got %q want %q; all=%v", i, actions[i], expected[i], actions)
		}
	}
}
