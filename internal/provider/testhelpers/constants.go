// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package testhelpers

import (
	"path/filepath"
)

const (
	// TmplPath defines the base path for template files.
	TmplPath = "./testdata/templates"
	// DataWorkTypesTmpl is the filename for the data.work_types Terraform template.
	DataWorkTypesTmpl = "data.work_types.tf.tmpl"
	// WorkTypeTmpl is the filename for the work_type Terraform template.
	WorkTypeTmpl = "work_type.tf.tmpl"
	// DataProjectTmpl is the filename for the data.projects Terraform template.
	DataProjectTmpl = "data.projects.tf.tmpl"
	// ProjectTmpl is the filename for the project Terraform template.
	ProjectTmpl = "project.tf.tmpl"
	// ProjectCatTmpl is the filename for the project_category Terraform template.
	ProjectCatTmpl = "project_category.tf.tmpl"
	// FieldTmpl is the filename for the field Terraform template.
	FieldTmpl = "field.tf.tmpl"
)

var (
	// DataWorkTypesTmplPath defines the file path for the data work types template based on the base template path.
	DataWorkTypesTmplPath = filepath.Join(TmplPath, DataWorkTypesTmpl)
	// WorkTypeTmplPath defines the file path for the work type template based on the base template path.
	WorkTypeTmplPath = filepath.Join(TmplPath, WorkTypeTmpl)

	// DataProjectTmplPath defines the file path for the data project template based on the base template path.
	DataProjectTmplPath = filepath.Join(TmplPath, DataProjectTmpl)
	// ProjectTmplPath defines the file path for the project template based on the base template path.
	ProjectTmplPath = filepath.Join(TmplPath, ProjectTmpl)

	// ProjectCatTmplPath defines the file path for the project category template based on the base template path.
	ProjectCatTmplPath = filepath.Join(TmplPath, ProjectCatTmpl)
	// FieldTmplPath defines the file path for the field template based on the base template path.
	FieldTmplPath = filepath.Join(TmplPath, FieldTmpl)

	// StandardWorkType represents the standard type of work, indicated with a value of 0.
	StandardWorkType = 0
	// SubtaskWorkType represents a subtask type of work, indicated with a value of -1.
	SubtaskWorkType = -1
)
