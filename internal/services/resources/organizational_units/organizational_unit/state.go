// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

// mapState applies usable directory observations while retaining explicitly managed hidden values.
func mapState(model *OrganizationalUnitResourceModel, unit *organizational_units.OrganizationalUnit) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if unit == nil || unit.GUID == nil || *unit.GUID == (shared.GUID{}) || unit.Name == nil {
		diagnostics.AddError("Incomplete OU observation", "The directory did not return the OU's GUID and name. Known identity and prior state are retained.")
		return diagnostics
	}
	id := unit.GUID.String()
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != id {
		diagnostics.AddError("Unexpected OU identity", "The read returned a different object GUID. Existing state is retained.")
		return diagnostics
	}
	model.ID = types.StringValue(id)
	model.Name = types.StringValue(*unit.Name)
	model.DistinguishedName = types.StringValue(unit.DistinguishedName)
	parent, ok := observedParent(unit.DistinguishedName)
	if !ok {
		diagnostics.AddError("Incomplete OU location", "The directory did not return a usable parent distinguished name.")
	} else if !strings.EqualFold(model.ParentDN.ValueString(), parent) {
		model.ParentDN = types.StringValue(parent)
	}
	ignored := organizational_units.OUProperties{}
	for _, field := range propertyBindings(model, &ignored) {
		observed := lifecycle.ObserveString(unit.Raw, field.ldap)
		diagnostics.Append(observed.Diagnostics...)
		switch {
		case observed.Usable:
			*field.value = observed.Value
		case !field.value.IsNull() && !field.value.IsUnknown():
			diagnostics.AddWarning(
				"OU property unavailable",
				"A managed property was hidden, absent or incomplete. Its previous value is retained; omission does not prove a clear.",
			)
		default:
			*field.value = types.StringNull()
		}
	}
	if unit.ProtectedFromAccidentalDeletion != nil {
		model.ProtectedFromAccidentalDeletion = types.BoolValue(*unit.ProtectedFromAccidentalDeletion)
	}
	return diagnostics
}

// observedParent locates the first unescaped RFC4514 separator after SDK validation, preserving wire spelling.
func observedParent(dn string) (string, bool) {
	return common.ParentDN(dn)
}

// persistState records known identity before returning a failure from later readback/workflow stages.
func persistState(ctx context.Context, state *tfsdk.State, identityState *tfsdk.ResourceIdentity, model *OrganizationalUnitResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	for _, field := range []*types.String{&model.ID, &model.DistinguishedName, &model.RecoveryDN} {
		if field.IsUnknown() {
			*field = types.StringNull()
		}
	}
	if model.ProtectedFromAccidentalDeletion.IsUnknown() {
		model.ProtectedFromAccidentalDeletion = types.BoolNull()
	}
	diagnostics.Append(state.Set(ctx, model)...)
	if identityState != nil && !model.ID.IsNull() {
		if canonical, err := identity.CanonicalGUID(model.ID.ValueString()); err == nil {
			diagnostics.Append(identityState.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
	return diagnostics
}
