// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// mapState applies only same-GUID usable observations, preserving prior properties when directory visibility is incomplete.
func mapState(model *GroupResourceModel, observed *groups.Group) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if observed == nil || observed.GUID == nil || *observed.GUID == (shared.GUID{}) || observed.Name == nil {
		diagnostics.AddError("Incomplete group identity", "A usable group GUID and name were not observed. Known state is retained.")
		return diagnostics
	}
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != observed.GUID.String() {
		diagnostics.AddError("Unexpected group identity", "The SDK returned another GUID. Existing identity and observations are retained.")
		return diagnostics
	}
	parent, valid := identity.ParentDN(observed.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete group location", "The SDK did not return a valid distinguished name and parent.")
		return diagnostics
	}
	model.ID, model.Name = types.StringValue(observed.GUID.String()), types.StringValue(*observed.Name)
	model.DistinguishedName = types.StringValue(observed.DistinguishedName)
	if !strings.EqualFold(parent, model.ParentDN.ValueString()) {
		model.ParentDN = types.StringValue(parent)
	}
	if observed.SID != nil {
		model.SID = types.StringValue(observed.SID.String())
	}
	ignored := groups.UpdateGroupRequest{}
	for _, field := range propertyBindings(model, &ignored) {
		value := lifecycle.ObserveString(observed.Raw, field.ldap)
		diagnostics.Append(value.Diagnostics...)
		switch {
		case value.Usable:
			*field.value = value.Value
		case field.value.IsUnknown() || field.value.IsNull():
			*field.value = types.StringNull()
		default:
			diagnostics.AddWarning(
				"Group property unavailable",
				"A managed property is hidden, absent or incomplete. Its prior value is retained; omission does not prove a clear.",
			)
		}
	}
	if observed.Category != nil {
		switch *observed.Category {
		case groups.GroupSecurity:
			model.Category = types.StringValue("security")
		case groups.GroupDistribution:
			model.Category = types.StringValue("distribution")
		default:
			diagnostics.AddError("Unknown group category", "Returned flags have no usable category. Prior category is retained; no flag mask is guessed.")
		}
	}
	if observed.Scope != nil {
		switch *observed.Scope {
		case groups.GroupGlobal:
			model.Scope = types.StringValue("global")
		case groups.GroupDomainLocal:
			model.Scope = types.StringValue("domain_local")
		case groups.GroupUniversal:
			model.Scope = types.StringValue("universal")
		default:
			diagnostics.AddError("Unknown group scope", "Returned groupType flags do not establish an unambiguous scope. Prior scope is retained.")
		}
	}
	if model.SAMAccountName.IsNull() || model.SAMAccountName.IsUnknown() || model.Category.IsNull() || model.Category.IsUnknown() || model.Scope.IsNull() ||
		model.Scope.IsUnknown() {
		diagnostics.AddError("Required group properties unavailable", "The SDK could not establish the group's account name, category and scope. No missing value is fabricated.")
	}
	return diagnostics
}

// persistState preserves the known identity and accepted observations even when later phases fail.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model *GroupResourceModel) diag.Diagnostics {
	ignored := groups.UpdateGroupRequest{}
	for _, field := range propertyBindings(model, &ignored) {
		if field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	for _, field := range []*types.String{&model.ID, &model.Name, &model.ParentDN, &model.Category, &model.Scope, &model.DistinguishedName, &model.SID, &model.RecoveryDN} {
		if field.IsUnknown() {
			*field = types.StringNull()
		}
	}
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() {
		canonical, err := identity.CanonicalGUID(model.ID.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid group state identity", "The retained group identity is not a nonzero GUID.")
		} else {
			diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
	return diagnostics
}
