// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	v3 "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// APIPayloadGetter defines a model capable of producing an API payload for create/update operations.
type APIPayloadGetter[P any] interface {
	GetAPIPayload(ctx context.Context) (createPayload P, diags diag.Diagnostics)
}

// TerraformTransformer defines a model that can map an API model into Terraform state.
type TerraformTransformer[R any] interface {
	IDer
	TransformToState(ctx context.Context, apiModel R) diag.Diagnostics
}

// IDer exposes a stable identifier accessor used by CRUD helpers and import.
type IDer interface {
	GetID() string
}

// ResourceTransformer combines API payload building and state transformation for a resource model.
type ResourceTransformer[P, R any] interface {
	APIPayloadGetter[P]
	TerraformTransformer[R]
}

// CreateResource is a generic helper that reads the plan, calls the API create function,
// transforms the API response into Terraform state, and sets it on the response.
func CreateResource[C, R any](ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse, resourceModel ResourceTransformer[C, R], createFunc func(ctx context.Context, newResource C) (R, *models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.Plan.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}
	newResource, diags := resourceModel.GetAPIPayload(ctx)

	response.Diagnostics.Append(diags...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := createFunc(ctx, newResource)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "create resource", apiResp, err, &response.Diagnostics, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses: []int{http.StatusOK, http.StatusCreated},
		IncludeBodySnippet: true,
	}) {
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)
}

// ReadResource is a generic helper that loads state, calls the API read function,
// handles 404 as not found by removing state, and updates state on success.
func ReadResource[R any](ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse, resourceModel TerraformTransformer[R], readFunc func(ctx context.Context, id string) (R, *models.ResponseScheme, error)) {

	response.Diagnostics.Append(request.State.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := readFunc(ctx, resourceModel.GetID())
	// If not found, remove state and return without error
	if HTTPStatusFromScheme(apiResp) == http.StatusNotFound {
		response.State.RemoveResource(ctx)
		return
	}
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "read resource", apiResp, err, &response.Diagnostics, &EnsureSuccessOrDiagOptions{
		IncludeBodySnippet: true,
	}) {
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)

}

// UpdateResource is a generic helper that reads the plan, calls the API update function,
// transforms the API response into Terraform state, and writes the updated state.
func UpdateResource[P, R any](ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse, resourceModel ResourceTransformer[P, R], updateFunc func(ctx context.Context, updatedResourceId string, updatedResource P) (R, *models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.Plan.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	updatedResource, diags := resourceModel.GetAPIPayload(ctx)

	response.Diagnostics.Append(diags...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := updateFunc(ctx, resourceModel.GetID(), updatedResource)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "update resource", apiResp, err, &response.Diagnostics, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses: []int{http.StatusOK, http.StatusNoContent},
		IncludeBodySnippet: true,
	}) {
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)

}

// DeleteResource is a generic helper that loads state, calls the API delete function,
// treats 404 as a successful no-op for idempotency, and returns diagnostics on failure.
func DeleteResource(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse, resourceModel IDer, deleteFunc func(ctx context.Context, id string) (*models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.State.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	apiResp, err := deleteFunc(ctx, resourceModel.GetID())
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "delete resource", apiResp, err, &response.Diagnostics, &EnsureSuccessOrDiagOptions{
		AcceptableStatuses:      []int{http.StatusOK, http.StatusNoContent},
		TreatDelete404AsSuccess: true,
		IncludeBodySnippet:      true,
	}) {
		return
	}
}

func ImportResource[R any](ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse, resourceModel TerraformTransformer[R], getFunc func(ctx context.Context, id string) (R, *models.ResponseScheme, error)) {

	resourceResp, apiResp, err := getFunc(ctx, request.ID)
	if !EnsureSuccessOrDiagFromSchemeWithOptions(ctx, "read imported resource", apiResp, err, &response.Diagnostics, &EnsureSuccessOrDiagOptions{
		IncludeBodySnippet: true,
	}) {
		return
	}
	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)
}

type baseJira struct {
	client           *v3.Client
	providerTimeouts opTimeouts
}
