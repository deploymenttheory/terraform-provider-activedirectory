// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account_links

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// ConstructPair validates both stable endpoint identifiers before any directory operation.
func ConstructPair(computer, account types.String) (shared.Identity, shared.Identity, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var selected [2]shared.Identity
	for index, value := range []types.String{computer, account} {
		if value.IsNull() || value.IsUnknown() {
			diagnostics.AddError(
				"Unknown association endpoint",
				"computer_guid and service_account_guid must both be known nonzero object GUIDs before reading or changing a link.",
			)
			continue
		}
		canonical, err := identity.CanonicalGUID(value.ValueString())
		if err != nil {
			diagnostics.AddError(
				"Invalid association endpoint",
				"computer_guid and service_account_guid must identify directory objects by nonzero GUID; names and DNs are not resource identities.",
			)
			continue
		}
		selected[index], _ = shared.IdentityByGUID(canonical)
	}
	return selected[0], selected[1], diagnostics
}
