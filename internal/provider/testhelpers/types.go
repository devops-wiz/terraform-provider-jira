// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package testhelpers

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// FakeTypeService provides a minimal test double for jira.TypeConnector used in unit tests.
// It exposes function fields so tests can customize behavior per case.
type FakeTypeService struct {
	CreateFn func(ctx context.Context, payload *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error)
	GetFn    func(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error)
	UpdateFn func(ctx context.Context, id string, payload *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error)
	DeleteFn func(ctx context.Context, id string) (*models.ResponseScheme, error)
}

func (f *FakeTypeService) Create(ctx context.Context, payload *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
	return f.CreateFn(ctx, payload)
}

func (f *FakeTypeService) Get(ctx context.Context, id string) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
	return f.GetFn(ctx, id)
}

func (f *FakeTypeService) Update(ctx context.Context, id string, payload *models.IssueTypePayloadScheme) (*models.IssueTypeScheme, *models.ResponseScheme, error) {
	return f.UpdateFn(ctx, id, payload)
}

func (f *FakeTypeService) Delete(ctx context.Context, id string) (*models.ResponseScheme, error) {
	return f.DeleteFn(ctx, id)
}

// FakeNetErr for testing ShouldRetry timeout path
type FakeNetErr struct{ timeout bool }

func (e FakeNetErr) Error() string   { return "fake timeout" }
func (e FakeNetErr) Timeout() bool   { return e.timeout }
func (e FakeNetErr) Temporary() bool { return true }

// NewFakeNetErr constructs a FakeNetErr with the provided timeout flag.
func NewFakeNetErr(timeout bool) FakeNetErr { return FakeNetErr{timeout: timeout} }

type ProjectCatTmplCfg struct {
	Name        string
	Description string
}

type ProjectTmplCfg struct {
	Key           string
	Name          string
	ProjectType   string
	LeadAccountID string
	Description   string
}

type DataProjectsCfg struct {
	ProjectResources []string
	DataName         string
	LookupBy         string
}

// fieldTemplateData represents the structure for storing field information such as name, type, and description.
type FieldTemplateData struct {
	Name        string
	FieldType   string
	Description string
}
