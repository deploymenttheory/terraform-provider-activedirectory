// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// constructTarget validates complete invocation inputs before any SDK call; ordinary writes use no version assertion.
func constructTarget(model AccountUnlockActionModel) (shared.Identity, diag.Diagnostics) {
	diagnostics := validateTarget(model, false)
	if diagnostics.HasError() {
		return shared.Identity{}, diagnostics
	}
	canonical, _ := identity.CanonicalGUID(model.ObjectGUID.ValueString())
	target, err := shared.IdentityByGUID(canonical)
	if err != nil {
		diagnostics.AddError("Invalid account unlock identity", "The SDK rejected the supplied GUID. No unlock was submitted.")
	}
	return target, diagnostics
}

// validateTarget checks known fields independently, deferring only genuinely unknown planning values.
func validateTarget(model AccountUnlockActionModel, allowUnknown bool) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if model.ObjectGUID.IsNull() || model.Server.IsNull() || (!allowUnknown && (model.ObjectGUID.IsUnknown() || model.Server.IsUnknown())) {
		diagnostics.AddError("Unknown account unlock target", "A known object_guid and server are required before invocation.")
	}
	if !model.ObjectGUID.IsUnknown() && !model.ObjectGUID.IsNull() {
		if _, err := identity.CanonicalGUID(model.ObjectGUID.ValueString()); err != nil {
			diagnostics.AddError("Invalid account unlock GUID", "object_guid must identify one existing account with a valid nonzero GUID.")
		}
	}
	if !model.Server.IsUnknown() && !model.Server.IsNull() {
		server := model.Server.ValueString()
		if server == "" || strings.ContainsAny(server, "/\\:@ \t\r\n") || !strings.Contains(server, ".") {
			diagnostics.AddError("Invalid account unlock server", "server must be the selected DC's DNS name without a URL scheme, port or whitespace.")
		}
	}
	return diagnostics
}
