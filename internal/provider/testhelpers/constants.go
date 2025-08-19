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
)

var (
	DataWorkTypesTmplPath = filepath.Join(TmplPath, DataWorkTypesTmpl)
	WorkTypeTmplPath      = filepath.Join(TmplPath, WorkTypeTmpl)

	StandardWorkType = 0
	SubtaskWorkType  = -1
)
