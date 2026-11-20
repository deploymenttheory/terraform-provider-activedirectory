// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account_links

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// QueryModel makes the computer scope and materialization bound explicit for collection and list discovery.
type QueryModel struct {
	ComputerGUID types.String `tfsdk:"computer_guid"`
	ResultLimit  types.Int64  `tfsdk:"result_limit"`
}

// ConstructQuery refuses unknown scope, non-GUID identities and unbounded materialization before SDK I/O.
func ConstructQuery(model QueryModel) (shared.Identity, int, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var computer shared.Identity
	canonical, err := identity.CanonicalGUID(model.ComputerGUID.ValueString())
	if err != nil || model.ComputerGUID.IsNull() || model.ComputerGUID.IsUnknown() {
		diagnostics.AddError("Invalid computer discovery scope", "computer_guid must be a known nonzero computer object GUID.")
	} else {
		computer, _ = shared.IdentityByGUID(canonical)
	}
	limit := int64(1000)
	if model.ResultLimit.IsUnknown() {
		diagnostics.AddError("Unknown association discovery bound", "result_limit must be known before querying direct links.")
	} else if !model.ResultLimit.IsNull() {
		limit = model.ResultLimit.ValueInt64()
	}
	if limit < 1 || limit > 10000 {
		diagnostics.AddError("Invalid association discovery bound", "result_limit must be between 1 and 10000; zero never enables an unbounded query.")
	}
	return computer, int(limit), diagnostics
}
