package provider

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OperationTimeoutsModel struct {
	Create types.String `tfsdk:"create"`
	Read   types.String `tfsdk:"read"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

type opTimeouts struct {
	Create time.Duration
	Read   time.Duration
	Update time.Duration
	Delete time.Duration
}
