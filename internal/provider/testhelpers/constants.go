// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package testhelpers

import (
	"path/filepath"
	"runtime"
)

const (
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

// TemplatesDir defines the base directory for template files.
const TemplatesDir = "templates"

// tmplPath builds a full path to a template inside TemplatesDir.
func tmplPath(name string) string {
	_, currentFile, _, _ := runtime.Caller(0)

	currentDir := filepath.Dir(currentFile)

	return filepath.Join(currentDir, TemplatesDir, name)
}

var (
	DataWorkTypesTmplPath = tmplPath(DataWorkTypesTmpl)
	WorkTypeTmplPath      = tmplPath(WorkTypeTmpl)
	DataProjectTmplPath   = tmplPath(DataProjectTmpl)
	ProjectTmplPath       = tmplPath(ProjectTmpl)
	ProjectCatTmplPath    = tmplPath(ProjectCatTmpl)
	FieldTmplPath         = tmplPath(FieldTmpl)
)

// Work type identifiers.
const (
	StandardWorkType = 0
	SubtaskWorkType  = -1
)
