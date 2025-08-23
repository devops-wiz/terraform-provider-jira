// Copyright (c) DevOps Wiz
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// generic readers (HCL over env, then default behavior per caller)
func readString(s types.String, env string) string {
	if !s.IsNull() && !s.IsUnknown() {
		return s.ValueString()
	}
	if env == "" {
		return ""
	}
	return os.Getenv(env)
}

func readInt64Default(v types.Int64, def int) int {
	if !v.IsNull() && !v.IsUnknown() {
		return int(v.ValueInt64())
	}
	return def
}

func readBoolDefault(v types.Bool, def bool) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	return def
}

// readStringWithAliases reads a string preferring the HCL value, then a canonical env var,
// then any number of alias env vars in order.
func readStringWithAliases(s types.String, canonical string, aliases ...string) string {
	// Prefer HCL, then canonical env
	if v := readString(s, canonical); v != "" {
		return v
	}
	// Fallback to aliases in provided order
	for _, a := range aliases {
		if a == "" {
			continue
		}
		if v := os.Getenv(a); v != "" {
			return v
		}
	}
	return ""
}
