package provider

import (
	"path/filepath"
)

const (
	tmplPath          = "./testdata/templates"
	dataWorkTypesTmpl = "data.work_types.tf.tmpl"
	workTypeTmpl      = "work_type.tf.tmpl"
)

var (
	dataWorkTypesTmplPath = filepath.Join(tmplPath, dataWorkTypesTmpl)
	workTypeTmplPath      = filepath.Join(tmplPath, workTypeTmpl)

	standardWorkType = 0
	subtaskWorkType  = -1
)
