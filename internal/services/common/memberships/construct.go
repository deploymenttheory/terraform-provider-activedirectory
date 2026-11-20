// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// ConstructPair validates both stable endpoint identifiers before any directory operation.
func ConstructPair(group, member types.String) (shared.Identity, shared.Identity, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var selected [2]shared.Identity
	for index, value := range []types.String{group, member} {
		if value.IsNull() || value.IsUnknown() {
			diagnostics.AddError("Unknown membership endpoint", "group_guid and member_guid must both be known nonzero object GUIDs before reading or changing a link.")
			continue
		}
		canonical, err := identity.CanonicalGUID(value.ValueString())
		if err != nil {
			diagnostics.AddError(
				"Invalid membership endpoint",
				"group_guid and member_guid must identify directory objects by nonzero GUID; names and DNs are not resource identities.",
			)
			continue
		}
		selected[index], _ = shared.IdentityByGUID(canonical)
	}
	return selected[0], selected[1], diagnostics
}
