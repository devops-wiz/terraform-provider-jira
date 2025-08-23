// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package testhelpers

import (
	"path/filepath"
)

const (
	TmplPath          = "./testdata/templates"
	DataWorkTypesTmpl = "data.work_types.tf.tmpl"
	WorkTypeTmpl      = "work_type.tf.tmpl"
	DataProjectTmpl   = "data.projects.tf.tmpl"
	ProjectTmpl       = "project.tf.tmpl"
)

var (
	DataWorkTypesTmplPath = filepath.Join(TmplPath, DataWorkTypesTmpl)
	WorkTypeTmplPath      = filepath.Join(TmplPath, WorkTypeTmpl)

	DataProjectTmplPath = filepath.Join(TmplPath, DataProjectTmpl)
	ProjectTmplPath     = filepath.Join(TmplPath, ProjectTmpl)

	StandardWorkType = 0
	SubtaskWorkType  = -1
)
