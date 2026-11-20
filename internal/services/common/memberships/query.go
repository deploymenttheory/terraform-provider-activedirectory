// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// QueryModel makes the group scope and materialization bound explicit for collection and list discovery.
type QueryModel struct {
	GroupGUID   types.String `tfsdk:"group_guid"`
	ResultLimit types.Int64  `tfsdk:"result_limit"`
}

// ConstructQuery refuses unknown scope, non-GUID identities and unbounded materialization before SDK I/O.
func ConstructQuery(model QueryModel) (shared.Identity, int, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var group shared.Identity
	canonical, err := identity.CanonicalGUID(model.GroupGUID.ValueString())
	if err != nil || model.GroupGUID.IsNull() || model.GroupGUID.IsUnknown() {
		diagnostics.AddError("Invalid group discovery scope", "group_guid must be a known nonzero group object GUID.")
	} else {
		group, _ = shared.IdentityByGUID(canonical)
	}
	limit := int64(1000)
	if model.ResultLimit.IsUnknown() {
		diagnostics.AddError("Unknown membership discovery bound", "result_limit must be known before querying direct links.")
	} else if !model.ResultLimit.IsNull() {
		limit = model.ResultLimit.ValueInt64()
	}
	if limit < 1 || limit > 10000 {
		diagnostics.AddError("Invalid membership discovery bound", "result_limit must be between 1 and 10000; zero never enables an unbounded query.")
	}
	return group, int(limit), diagnostics
}
