package provider

import (
	"context"
	"fmt"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ApiPayloadGetter[P any] interface {
	GetApiPayload(ctx context.Context) (createPayload P, diags diag.Diagnostics)
}

type TerraformTransformer[R any] interface {
	IDer
	TransformToState(ctx context.Context, apiModel R) diag.Diagnostics
}

type IDer interface {
	GetID() string
}

type ResourceTransformer[P, R any] interface {
	ApiPayloadGetter[P]
	TerraformTransformer[R]
}

func CreateResource[C, R any](ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse, resourceModel ResourceTransformer[C, R], createFunc func(ctx context.Context, newResource C) (R, *models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.Plan.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}
	newResource, diags := resourceModel.GetApiPayload(ctx)

	response.Diagnostics.Append(diags...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := createFunc(ctx, newResource)

	if err != nil {
		response.Diagnostics.AddError("Error creating resource", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if apiResp.StatusCode != 201 {
		response.Diagnostics.AddError("Error creating resource", fmt.Sprintf("Error: %s", apiResp.Body))
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)
}

func ReadResource[R any](ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse, resourceModel TerraformTransformer[R], readFunc func(ctx context.Context, id string) (R, *models.ResponseScheme, error)) {

	response.Diagnostics.Append(request.State.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := readFunc(ctx, resourceModel.GetID())

	if err != nil {
		response.Diagnostics.AddError("Error reading resource", fmt.Sprintf("Error:\n%s", err.Error()))
		return
	}

	if apiResp.StatusCode != 200 {
		response.Diagnostics.AddError("Error reading resource", fmt.Sprintf("Error:\n%s", apiResp.Body))
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)

}

func UpdateResource[P, R any](ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse, resourceModel ResourceTransformer[P, R], updateFunc func(ctx context.Context, updatedResourceId string, updatedResource P) (R, *models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.Plan.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	updatedResource, diags := resourceModel.GetApiPayload(ctx)

	response.Diagnostics.Append(diags...)

	if response.Diagnostics.HasError() {
		return
	}

	resourceResp, apiResp, err := updateFunc(ctx, resourceModel.GetID(), updatedResource)

	if err != nil {
		response.Diagnostics.AddError("Error updating resource", fmt.Sprintf("Error:\n%s", err))
		return
	}

	if apiResp.StatusCode != 200 {
		response.Diagnostics.AddError("Error updating resource", fmt.Sprintf("Error:\n%s", apiResp.Body))
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)

}

func DeleteResource(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse, resourceModel IDer, deleteFunc func(ctx context.Context, id string) (*models.ResponseScheme, error)) {
	response.Diagnostics.Append(request.State.Get(ctx, resourceModel)...)

	if response.Diagnostics.HasError() {
		return
	}

	apiResp, err := deleteFunc(ctx, resourceModel.GetID())
	if err != nil {
		response.Diagnostics.AddError("Error deleting resource", fmt.Sprintf("Error:\n%s", err.Error()))
		return
	}

	if apiResp.StatusCode != 204 {
		response.Diagnostics.AddError("Error deleting resource", fmt.Sprintf("Error:\n%s", apiResp.Body))
		return
	}
}

func ImportResource[R any](ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse, resourceModel TerraformTransformer[R], getFunc func(ctx context.Context, id string) (R, *models.ResponseScheme, error)) {

	resourceResp, apiResp, err := getFunc(ctx, request.ID)

	if err != nil {
		response.Diagnostics.AddError("Error reading imported resource", fmt.Sprintf("Error:\n%s", err.Error()))
		return
	}

	if apiResp.StatusCode != 200 {
		response.Diagnostics.AddError("Error reading imported resource", fmt.Sprintf("Error:\n%s", apiResp.Body))
		return
	}

	response.Diagnostics.Append(resourceModel.TransformToState(ctx, resourceResp)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, resourceModel)...)
}
