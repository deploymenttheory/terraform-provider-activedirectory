// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

type boolProperty struct {
	name   string
	value  *types.Bool
	target **bool
}

// boolProperties binds individual account flags without exposing a raw userAccountControl replacement mask.
func boolProperties(model *UserResourceModel, properties *users.UserProperties) []boolProperty {
	return []boolProperty{
		{"enabled", &model.Enabled, &properties.Enabled},
		{"password_not_required", &model.PasswordNotRequired, &properties.PasswordNotRequired},
		{"password_never_expires", &model.PasswordNeverExpires, &properties.PasswordNeverExpires},
		{"smartcard_logon_required", &model.SmartcardLogonRequired, &properties.SmartcardLogonRequired},
	}
}

// mapState applies returned values only, retaining prior managed values when attributes or account flags are hidden.
func mapState(model *UserResourceModel, observed *users.User) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if observed == nil || observed.GUID == nil || *observed.GUID == (shared.GUID{}) || observed.Name == nil {
		diagnostics.AddError("Incomplete user identity", "The SDK did not observe a usable GUID and name. Known state is retained.")
		return diagnostics
	}
	id := observed.GUID.String()
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != id {
		diagnostics.AddError("Unexpected user identity", "The SDK returned another GUID. Existing identity and state are retained.")
		return diagnostics
	}
	parent, valid := identity.ParentDN(observed.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete user location", "The SDK returned an invalid distinguished name. Existing identity and state are retained.")
		return diagnostics
	}
	model.ID, model.Name = types.StringValue(id), types.StringValue(*observed.Name)
	model.DistinguishedName = types.StringValue(observed.DistinguishedName)
	if !strings.EqualFold(model.ParentDN.ValueString(), parent) {
		model.ParentDN = types.StringValue(parent)
	}
	if observed.SID != nil {
		model.SID = types.StringValue(observed.SID.String())
	}
	classes, classState := observed.Raw.Attribute("objectClass")
	if classState == shared.AttributePresent {
		for _, class := range classes {
			if strings.EqualFold(string(class), "inetOrgPerson") {
				model.ObjectClass = types.StringValue("inetOrgPerson")
				break
			}
			if strings.EqualFold(string(class), "user") {
				model.ObjectClass = types.StringValue("user")
			}
		}
	}
	ignored := users.UserProperties{}
	for _, field := range textProperties(model, &ignored) {
		value := lifecycle.ObserveString(observed.Raw, field.ldap)
		diagnostics.Append(value.Diagnostics...)
		switch {
		case value.Usable:
			*field.value = value.Value
		case !field.value.IsNull() && !field.value.IsUnknown():
			diagnostics.AddWarning("User property unavailable", "A managed property is hidden, absent or incomplete. Its prior value is retained; omission does not prove a clear.")
		default:
			*field.value = types.StringNull()
		}
	}
	for _, field := range boolProperties(model, &observed.UserProperties) {
		if *field.target != nil {
			*field.value = types.BoolValue(**field.target)
		} else if field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	if observed.AccountExpires != nil {
		if *observed.AccountExpires > math.MaxInt64 {
			diagnostics.AddError("Unsupported user expiration value", "accountExpires exceeded the supported nonnegative FILETIME range; prior state is retained.")
		} else {
			model.AccountExpires = types.Int64Value(int64(*observed.AccountExpires))
		}
	}
	return diagnostics
}

// persistState erases write-only input before serialization, while preserving the acknowledged version and stable import identity.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model *UserResourceModel) diag.Diagnostics {
	model.PasswordWO = types.StringNull()
	ignored := users.UserProperties{}
	for _, field := range textProperties(model, &ignored) {
		if field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	for _, field := range boolProperties(model, &ignored) {
		if field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	for _, field := range []*types.String{&model.ID, &model.Name, &model.ParentDN, &model.ObjectClass, &model.DistinguishedName, &model.SID, &model.RecoveryDN} {
		if field.IsUnknown() {
			*field = types.StringNull()
		}
	}
	if model.AccountExpires.IsUnknown() {
		model.AccountExpires = types.Int64Null()
	}
	if model.PasswordWOVersion.IsUnknown() || model.PasswordWOVersion.IsNull() {
		model.PasswordWOVersion = types.Int64Value(0)
	}
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() {
		canonical, err := identity.CanonicalGUID(model.ID.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid user state identity", "The stored object identity is not a valid GUID.")
		} else {
			diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
	return diagnostics
}
