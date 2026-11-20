// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// ConstructLookup selects exactly one explicit stable GUID or distinguished name before any SDK read.
func ConstructLookup(guid, dn types.String) (shared.Identity, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if guid.IsUnknown() || dn.IsUnknown() || guid.IsNull() == dn.IsNull() {
		diagnostics.AddError("Invalid account settings lookup", "Supply exactly one known object_guid or dn.")
		return shared.Identity{}, diagnostics
	}
	var target shared.Identity
	var err error
	if !guid.IsNull() {
		var canonical string
		canonical, err = identity.CanonicalGUID(guid.ValueString())
		if err == nil {
			target, err = shared.IdentityByGUID(canonical)
		}
	} else {
		target, err = shared.IdentityByDN(dn.ValueString())
	}
	if err != nil {
		diagnostics.AddError("Invalid account settings identity", "Use a nonzero object GUID or a valid explicit distinguished name.")
	}
	return target, diagnostics
}
