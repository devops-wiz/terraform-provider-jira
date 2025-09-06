// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Function type aliases used by CRUDHooks for clarity and reuse.
//
// Purpose
// - Keep CRUDHooks signatures concise and intention‑revealing (builder, create/read/update/delete, mapper).
// - Centralize call signatures so future changes don’t ripple across all resources.
// - Leverage the constraints below to catch type mismatches at compile time.
//
// Usage pattern
// - Resources implement these functions via small closures (for get/set state) and concrete client methods.
// - For mapping, prefer reusing a shared MapToState function per resource (e.g., map*SchemeToModel).

// PayloadBuilderFunc builds the API payload (TPayload) from the planned Terraform state (TState).
// Return diagnostics for validation or value derivation errors encountered during build.
type PayloadBuilderFunc[TState StateConstraint, TPayload PayloadConstraint] func(ctx context.Context, st *TState) (TPayload, diag.Diagnostics)

// CreateFunc invokes the concrete API Create call and returns the created API model (TAPI).
// The models.ResponseScheme is threaded for Ensure* handling (status/body diagnostics).
type CreateFunc[TPayload PayloadConstraint, TAPI APIConstraint] func(ctx context.Context, p TPayload) (api TAPI, apiResp *models.ResponseScheme, err error)

// ReadFunc invokes the concrete API Read/Get call using a stable identifier and returns TAPI.
type ReadFunc[TAPI APIConstraint] func(ctx context.Context, id string) (api TAPI, apiResp *models.ResponseScheme, err error)

// UpdateFunc invokes the concrete API Update call with payload and returns the refreshed TAPI.
// Some services may issue Update → Get to return a full object; both patterns are supported.
type UpdateFunc[TPayload PayloadConstraint, TAPI APIConstraint] func(ctx context.Context, id string, p TPayload) (api TAPI, apiResp *models.ResponseScheme, err error)

// DeleteFunc invokes the concrete API Delete call; Ensure* handles status evaluation and 404 semantics.
type DeleteFunc func(ctx context.Context, id string) (apiResp *models.ResponseScheme, err error)

// ExtractIDFunc returns the stable identifier from the current state (typically id string).
type ExtractIDFunc[TState StateConstraint] func(st *TState) string

// MapToStateFunc maps the API model (TAPI) into the Terraform state model (TState).
// Return diagnostics when mapping encounters unexpected values or schema mismatches.
type MapToStateFunc[TState StateConstraint, TAPI APIConstraint] func(ctx context.Context, api TAPI, st *TState) diag.Diagnostics

// PostAPIHook is an optional extra step after Create to fetch the full TAPI model.
// Use this when Create returns a partial object and a subsequent Get is needed.
type PostAPIHook[TState StateConstraint, TAPI APIConstraint] func(ctx context.Context, api TAPI, st *TState) (apiOut TAPI, apiResp *models.ResponseScheme, err error)

// Generic type constraints restricted to available provider types.
//
// What these do
// - StateConstraint limits TState to Terraform state models defined by this provider.
// - PayloadConstraint limits TPayload to concrete go‑atlassian payload types the provider sends.
// - APIConstraint limits TAPI to the concrete go‑atlassian models returned by the client.
//
// Why constrain?
// - Compile‑time safety across resources (you can’t wire a project payload into a work type resource).
// - Better IDE help and autocomplete.
// - Clear “extension points” when adding a new resource.
//
// How to extend
// - Add your new types to the relevant union below.
// - Implement hooks() in the new resource that uses those types.

// StateConstraint enumerates the Terraform state models supported by the CRUD runner.
type StateConstraint interface {
	projectResourceModel |
		workTypeResourceModel |
		projectCategoryResourceModel |
		fieldResourceModel
}

// PayloadConstraint enumerates the supported go‑atlassian payload types used in Create/Update.
type PayloadConstraint interface {
	*models.ProjectPayloadScheme |
		*models.IssueTypePayloadScheme |
		*models.ProjectCategoryPayloadScheme |
		*models.CustomFieldScheme
}

// APIConstraint enumerates the supported go‑atlassian API models returned by service calls.
type APIConstraint interface {
	*models.ProjectScheme |
		*models.IssueTypeScheme |
		*models.ProjectCategoryScheme |
		*models.IssueFieldScheme
}

// CRUDHooks defines per‑resource behavior consumed by the generic runner.
//
// Provide only the resource‑specific logic here; the runner coordinates the flow.
//
// Required
// - BuildPayload: Construct the API payload from planned state (Create/Update).
// - APICreate/APIRead/APIUpdate/APIDelete: Call the concrete go‑atlassian client methods.
// - ExtractID: Return the stable identifier from state (used for Read/Update/Delete).
// - MapToState: Map the API model to Terraform state (also used by Import).
//
// Optional
// - PostCreateRead: If Create returns a partial object, fetch the full model (Create → Get).
// - Acceptable*Statuses: Override HTTP statuses that should be treated as success per operation.
// - TreatDelete404AsSuccess: Make Delete idempotent by treating 404 as success.
//
// Typical wiring
// - Define a hooks() method on each resource that returns CRUDHooks with all fields set.
// - Use ensureWith(&resp.Diagnostics) (see shared_helpers.go) to bind diagnostics in Do* calls.
// - Use shared mappers (map*SchemeToModel) for MapToState to keep behavior consistent.
type CRUDHooks[TState StateConstraint, TPayload PayloadConstraint, TAPI APIConstraint] struct {
	// Required
	BuildPayload PayloadBuilderFunc[TState, TPayload]

	// API calls
	APICreate CreateFunc[TPayload, TAPI]
	APIRead   ReadFunc[TAPI]
	APIUpdate UpdateFunc[TPayload, TAPI]
	APIDelete DeleteFunc

	// State helpers
	ExtractID  ExtractIDFunc[TState]
	MapToState MapToStateFunc[TState, TAPI]

	// Optional hook for Create flows that need a follow-up read
	PostCreate PostAPIHook[TState, TAPI]
	PostRead   PostAPIHook[TState, TAPI]
	PostUpdate PostAPIHook[TState, TAPI]

	// Per-operation status options
	AcceptableCreateStatuses []int
	AcceptableUpdateStatuses []int
	AcceptableDeleteStatuses []int
	TreatDelete404AsSuccess  bool
}

// orDefaultStatuses returns the provided statuses when non‑empty,
// otherwise falls back to the supplied defaults. Used by Ensure* evaluation.
func orDefaultStatuses(got []int, def ...int) []int {
	if len(got) > 0 {
		return got
	}
	return def
}

// CRUDRunner coordinates the CRUD lifecycle using the per‑resource CRUDHooks.
// It is generic over TState (Terraform model), TPayload (API payload), and TAPI (API model).
type CRUDRunner[TState StateConstraint, TPayload PayloadConstraint, TAPI APIConstraint] struct {
	// hooks holds the per‑resource implementation details consumed by the runner.
	hooks CRUDHooks[TState, TPayload, TAPI]
}

// -----------------------------
// List/Search generic contracts
// -----------------------------

// ListFunc lists API items; may include internal pagination.
// Prefer ListPage for server-side pagination when available.
// Kept for compatibility with data sources and tests.
// TAPIList is independent from TAPI of CRUDRunner and constrained by APIListConstraint.
type ListFunc[TAPIList APIListConstraint] func(ctx context.Context) ([]TAPIList, diag.Diagnostics)

// FilterFunc filters an API item; return true to keep.
type FilterFunc[TAPIList APIListConstraint] func(ctx context.Context, item TAPIList) bool

// KeyOfFunc returns a stable key for an item (e.g., ID as string).
type KeyOfFunc[TAPIList APIListConstraint] func(item TAPIList) string

// MapToOutFunc maps an API item to the Terraform object model.
type MapToOutFunc[TAPIList APIListConstraint, TOut OutModelConstraint] func(ctx context.Context, item TAPIList) (TOut, diag.Diagnostics)

// AttrTypesFunc returns the attribute types for TOut’s ObjectType; used by callers when encoding.
type AttrTypesFunc func() map[string]attr.Type

// ListPageFunc pages through list results.
// startAt: zero-based offset, max: desired page size.
// Return items, whether this is the last page, and diagnostics.
type ListPageFunc[TAPIList APIListConstraint] func(ctx context.Context, startAt, max int) (items []TAPIList, isLast bool, d diag.Diagnostics)

// APIListConstraint enumerates API models that appear in lists.
type APIListConstraint interface {
	*models.ProjectScheme | *models.ProjectCategoryScheme | *models.IssueTypeScheme
}

// OutModelConstraint enumerates Terraform object models used as list outputs.
type OutModelConstraint interface {
	projectResourceModel | projectCategoryResourceModel | workTypeResourceModel
}

// ListHooks defines list-to-map helpers for data sources and utilities.
// TAPIList: API model per item.
// TOut: Terraform object model per item.
type ListHooks[TAPIList APIListConstraint, TOut OutModelConstraint] struct {
	List      ListFunc[TAPIList]
	ListPage  ListPageFunc[TAPIList]
	Filter    FilterFunc[TAPIList]
	KeyOf     KeyOfFunc[TAPIList]
	MapToOut  MapToOutFunc[TAPIList, TOut]
	AttrTypes AttrTypesFunc
}

// ListOptions configures DoListToMapWithLimit behavior.
type ListOptions struct {
	MaxItems       int
	WarnThreshold  int
	PreallocCap    int
	RespectContext bool
}

// NewCRUDRunner constructs a CRUDRunner bound to the provided hooks.
// Typical usage: runner := NewCRUDRunner(r.hooks())
func NewCRUDRunner[TState StateConstraint, TPayload PayloadConstraint, TAPI APIConstraint](hooks CRUDHooks[TState, TPayload, TAPI]) CRUDRunner[TState, TPayload, TAPI] {
	return CRUDRunner[TState, TPayload, TAPI]{hooks: hooks}
}

// runPostHook runs an optional post-API hook (create/read/update) with shared ensure handling.
func (r CRUDRunner[TState, TPayload, TAPI]) runPostHook(
	ctx context.Context,
	label string,
	hook PostAPIHook[TState, TAPI],
	api TAPI,
	st *TState,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
) (TAPI, bool) {
	if hook == nil {
		return api, true
	}
	api2, rs, err := hook(ctx, api, st)
	if !ensure(ctx, label, rs, err, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
		var zero TAPI
		return zero, false
	}
	return api2, true
}

// mapAndSetState performs the MapToState + setState sequence and returns accumulated diagnostics.
func (r CRUDRunner[TState, TPayload, TAPI]) mapAndSetState(
	ctx context.Context,
	api TAPI,
	st *TState,
	setState func(ctx context.Context, src *TState) diag.Diagnostics,
) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(r.hooks.MapToState(ctx, api, st)...)
	if diags.HasError() {
		return diags
	}
	diags.Append(setState(ctx, st)...)
	return diags
}

// ensureCreateOK evaluates success for create operations using AcceptableCreateStatuses.
func (r CRUDRunner[TState, TPayload, TAPI]) ensureCreateOK(
	ctx context.Context,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
	rs *models.ResponseScheme,
	err error,
) bool {
	return ensure(ctx, "create resource", rs, err, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses: orDefaultStatuses(r.hooks.AcceptableCreateStatuses, http.StatusOK, http.StatusCreated),
		IncludeBodySnippet: true,
	})
}

// ensureReadOK evaluates success for read operations.
func (r CRUDRunner[TState, TPayload, TAPI]) ensureReadOK(
	ctx context.Context,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
	rs *models.ResponseScheme,
	err error,
) bool {
	return ensure(ctx, "read resource", rs, err, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true})
}

// ensureUpdateOK evaluates success for update operations using AcceptableUpdateStatuses.
func (r CRUDRunner[TState, TPayload, TAPI]) ensureUpdateOK(
	ctx context.Context,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
	rs *models.ResponseScheme,
	err error,
) bool {
	return ensure(ctx, "update resource", rs, err, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses: orDefaultStatuses(r.hooks.AcceptableUpdateStatuses, http.StatusOK, http.StatusNoContent),
		IncludeBodySnippet: true,
	})
}

// ensureDeleteOK evaluates success for delete operations using AcceptableDeleteStatuses and 404 idempotency.
func (r CRUDRunner[TState, TPayload, TAPI]) ensureDeleteOK(
	ctx context.Context,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
	rs *models.ResponseScheme,
	err error,
) bool {
	return ensure(ctx, "delete resource", rs, err, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses:      orDefaultStatuses(r.hooks.AcceptableDeleteStatuses, http.StatusOK, http.StatusNoContent),
		TreatDelete404AsSuccess: r.hooks.TreatDelete404AsSuccess,
		IncludeBodySnippet:      true,
	})
}

// handleRead404 checks response status and triggers remove() when Not Found.
// Returns true if 404 was handled and the caller should stop further processing.
func (r CRUDRunner[TState, TPayload, TAPI]) handleRead404(
	ctx context.Context,
	rs *models.ResponseScheme,
	httpStatus func(*models.ResponseScheme) int,
	remove func(ctx context.Context),
) bool {
	if httpStatus(rs) == http.StatusNotFound {
		remove(ctx)
		return true
	}
	return false
}

// DoCreate orchestrates the Create lifecycle:
//
// Steps
// 1) getPlan: Read the Terraform planned state into TState.
// 2) BuildPayload: Build the API payload (TPayload) from TState.
// 3) APICreate: Call the service to create the resource.
// 4) PostCreate (optional): Fetch the full model if Create returns a partial object.
// 5) MapToState: Map the API model (TAPI) back into TState.
// 6) setState: Persist the final state back to Terraform.
//
// ensure
// - Pass ensureWith(&resp.Diagnostics) to bind resource diagnostics to the Ensure helper.
// - Configure AcceptableCreateStatuses on hooks if non‑default statuses should be treated as success.
func (r CRUDRunner[TState, TPayload, TAPI]) DoCreate(
	ctx context.Context,
	getPlan func(ctx context.Context, dst *TState) diag.Diagnostics,
	setState func(ctx context.Context, src *TState) diag.Diagnostics,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	var st TState

	// Optional Create support per hooks
	if r.hooks.APICreate == nil || r.hooks.BuildPayload == nil {
		diags.AddError("create unsupported", "This resource does not support create operation.")
		return diags
	}

	// 1) Read planned state
	if d := getPlan(ctx, &st); d.HasError() {
		return d
	}

	// 2) Build payload
	payload, d2 := r.hooks.BuildPayload(ctx, &st)
	diags.Append(d2...)
	if diags.HasError() {
		return diags
	}

	// 3) API create
	api, rs, err := r.hooks.APICreate(ctx, payload)
	if !r.ensureCreateOK(ctx, ensure, rs, err) {
		return diags
	}

	// 4) Optional post-create read
	api, ok := r.runPostHook(ctx, "post-create hook", r.hooks.PostCreate, api, &st, ensure)
	if !ok {
		return diags
	}

	// 5) Map and set state
	mapped := r.mapAndSetState(ctx, api, &st, setState)
	diags.Append(mapped...)
	return diags
}

// DoRead refreshes state from the remote API.
//
// Behavior
// - Reads current state (for ID) via getState.
// - Invokes APIRead; if httpStatus(response) == 404, remove() is called to drop the resource from state.
// - Otherwise maps API → state (MapToState) and writes it using setState.
// - Any HTTP/transport errors are reported via ensure.
func (r CRUDRunner[TState, TPayload, TAPI]) DoRead(
	ctx context.Context,
	getState func(ctx context.Context, dst *TState) diag.Diagnostics,
	setState func(ctx context.Context, src *TState) diag.Diagnostics,
	remove func(ctx context.Context),
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
	httpStatus func(*models.ResponseScheme) int,
) diag.Diagnostics {
	var diags diag.Diagnostics
	var st TState

	// Required hooks for read/import semantics
	if r.hooks.APIRead == nil || r.hooks.ExtractID == nil || r.hooks.MapToState == nil {
		diags.AddError("read unsupported", "This resource does not support read operation.")
		return diags
	}

	// 1) Read current state (for ID)
	if d := getState(ctx, &st); d.HasError() {
		return d
	}
	id := r.hooks.ExtractID(&st)

	// 2) API read (404 removal)
	api, rs, err := r.hooks.APIRead(ctx, id)
	if r.handleRead404(ctx, rs, httpStatus, remove) {
		return diags
	}
	if !r.ensureReadOK(ctx, ensure, rs, err) {
		return diags
	}

	// 3) Post-read hook
	api, ok := r.runPostHook(ctx, "post-read hook", r.hooks.PostRead, api, &st, ensure)
	if !ok {
		return diags
	}

	// 4) Map and set state
	mapped := r.mapAndSetState(ctx, api, &st, setState)
	diags.Append(mapped...)
	return diags
}

// DoUpdate applies changes to the remote API and updates state.
//
// Steps
// - getPlan → BuildPayload → APIUpdate
// - MapToState → setState
//
// Notes
// - AcceptableUpdateStatuses default to 200/204; override on hooks if needed.
// - Use ensureWith(&resp.Diagnostics) for consistent error/HTTP handling.
func (r CRUDRunner[TState, TPayload, TAPI]) DoUpdate(
	ctx context.Context,
	getPlan func(ctx context.Context, dst *TState) diag.Diagnostics,
	setState func(ctx context.Context, src *TState) diag.Diagnostics,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	var st TState

	// Optional Update support per hooks
	if r.hooks.APIUpdate == nil || r.hooks.BuildPayload == nil || r.hooks.ExtractID == nil {
		diags.AddError("update unsupported", "This resource does not support update operation.")
		return diags
	}

	// 1) Read planned state (for ID)
	if d := getPlan(ctx, &st); d.HasError() {
		return d
	}
	id := r.hooks.ExtractID(&st)

	// 2) Build payload
	payload, d2 := r.hooks.BuildPayload(ctx, &st)
	diags.Append(d2...)
	if diags.HasError() {
		return diags
	}

	// 3) API update
	api, rs, err := r.hooks.APIUpdate(ctx, id, payload)
	if !r.ensureUpdateOK(ctx, ensure, rs, err) {
		return diags
	}

	// 4) Post-update hook
	api, ok := r.runPostHook(ctx, "post-update hook", r.hooks.PostUpdate, api, &st, ensure)
	if !ok {
		return diags
	}

	// 5) Map and set state
	mapped := r.mapAndSetState(ctx, api, &st, setState)
	diags.Append(mapped...)
	return diags
}

// DoDelete removes the resource remotely.
//
// Steps
//   - getState to obtain the ID
//   - APIDelete call
//   - ensure handles success/HTTP codes; if TreatDelete404AsSuccess is set on hooks,
//     a 404 is treated as success for idempotent destroys.
func (r CRUDRunner[TState, TPayload, TAPI]) DoDelete(
	ctx context.Context,
	getState func(ctx context.Context, dst *TState) diag.Diagnostics,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	var st TState

	// Optional Delete support per hooks
	if r.hooks.APIDelete == nil || r.hooks.ExtractID == nil {
		diags.AddError("delete unsupported", "This resource does not support delete operation.")
		return diags
	}

	// 1) Read current state (for ID)
	if d := getState(ctx, &st); d.HasError() {
		return d
	}
	id := r.hooks.ExtractID(&st)

	// 2) API delete
	rs, err := r.hooks.APIDelete(ctx, id)
	if !r.ensureDeleteOK(ctx, ensure, rs, err) {
		return diags
	}
	return diags
}

// DoImport mirrors Read using an arbitrary import identifier.
//
// Steps
// - APIRead fetches the remote model using the import ID.
// - MapToState converts the API model into Terraform state.
// - setState persists the state.
//
// Guidance
// - Pass ensureWith(&response.Diagnostics) for ensure.
// - Reuse your resource’s MapToState hook implementation for consistency.
func (r CRUDRunner[TState, TPayload, TAPI]) DoImport(
	ctx context.Context,
	id string,
	setState func(ctx context.Context, src *TState) diag.Diagnostics,
	ensure func(ctx context.Context, action string, resp *models.ResponseScheme, err error, opts *EnsureSuccessOrDiagOptions) bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	var st TState

	// Optional Import support per hooks
	if r.hooks.APIRead == nil || r.hooks.MapToState == nil {
		diags.AddError("import unsupported", "This resource does not support import operation.")
		return diags
	}

	api, rs, err := r.hooks.APIRead(ctx, id)
	if !ensure(ctx, "read imported resource", rs, err, &EnsureSuccessOrDiagOptions{IncludeBodySnippet: true}) {
		return diags
	}

	api, ok := r.runPostHook(ctx, "post-read on import hook", r.hooks.PostRead, api, &st, ensure)
	if !ok {
		return diags
	}

	diags.Append(r.hooks.MapToState(ctx, api, &st)...)
	if diags.HasError() {
		return diags
	}
	diags.Append(setState(ctx, &st)...)
	return diags
}

// doListToMapCore is the generic implementation backing typed CRUDRunner list methods.
func doListToMapCore[TAPIList APIListConstraint, TOut OutModelConstraint](
	ctx context.Context,
	h ListHooks[TAPIList, TOut],
	opts ListOptions,
) (map[string]TOut, diag.Diagnostics) {
	const (
		pageSizeDefault  = 200
		checkCancelEvery = 1000
	)
	var diags diag.Diagnostics

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
			diags.AddWarning("large result set", "number of items kept reached threshold: "+strconv.Itoa(count))
		}
	}
	warnCapped := func(max int) {
		diags.AddWarning("result capped", "maximum items reached; result truncated at "+strconv.Itoa(max))
	}

	kept := 0
	processed := 0

	processItems := func(items []TAPIList, result map[string]TOut) bool {
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
			result[k] = obj
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

	startAt := 0
	max := pageSizeDefault
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

// Typed CRUDRunner list methods for current provider use cases.
func (r CRUDRunner[TState, TPayload, TAPI]) DoListIssueTypes(
	ctx context.Context,
	h ListHooks[*models.IssueTypeScheme, workTypeResourceModel],
) (map[string]workTypeResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, ListOptions{})
}

func (r CRUDRunner[TState, TPayload, TAPI]) DoListIssueTypesWithLimit(
	ctx context.Context,
	h ListHooks[*models.IssueTypeScheme, workTypeResourceModel],
	opts ListOptions,
) (map[string]workTypeResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, opts)
}

func (r CRUDRunner[TState, TPayload, TAPI]) DoListProjects(
	ctx context.Context,
	h ListHooks[*models.ProjectScheme, projectResourceModel],
) (map[string]projectResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, ListOptions{})
}

func (r CRUDRunner[TState, TPayload, TAPI]) DoListProjectsWithLimit(
	ctx context.Context,
	h ListHooks[*models.ProjectScheme, projectResourceModel],
	opts ListOptions,
) (map[string]projectResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, opts)
}

func (r CRUDRunner[TState, TPayload, TAPI]) DoListProjectCategories(
	ctx context.Context,
	h ListHooks[*models.ProjectCategoryScheme, projectCategoryResourceModel],
) (map[string]projectCategoryResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, ListOptions{})
}

func (r CRUDRunner[TState, TPayload, TAPI]) DoListProjectCategoriesWithLimit(
	ctx context.Context,
	h ListHooks[*models.ProjectCategoryScheme, projectCategoryResourceModel],
	opts ListOptions,
) (map[string]projectCategoryResourceModel, diag.Diagnostics) {
	return doListToMapCore(ctx, h, opts)
}
