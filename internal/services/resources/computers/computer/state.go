// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// controlObservation maps stored settings and explicit pwdLastSet observation without inventing hidden flags or host join status.
func controlObservation(state accounts.AccountControlState) accounts.AccountControlPatch {
	control := accounts.AccountControlPatch{
		Enabled: state.Enabled, AccountNotDelegated: state.AccountNotDelegated, AllowReversiblePasswordEncryption: state.AllowReversiblePasswordEncryption,
		DoesNotRequirePreAuth: state.DoesNotRequirePreAuth, HomeDirectoryRequired: state.HomeDirectoryRequired, MNSLogonAccount: state.MNSLogonAccount,
		PasswordNeverExpires: state.PasswordNeverExpires, PasswordNotRequired: state.PasswordNotRequired, SmartcardLogonRequired: state.SmartcardLogonRequired,
		TrustedForDelegation: state.TrustedForDelegation, TrustedToAuthForDelegation: state.TrustedToAuthForDelegation, UseDESKeyOnly: state.UseDESKeyOnly,
	}
	if state.PasswordLastSet != nil {
		value := *state.PasswordLastSet == 0
		control.ChangePasswordAtLogon = &value
	}
	return control
}

// observedEncryptionTypes returns the typed low-five-bit selection while leaving compound identity and unknown bits separate.
func observedEncryptionTypes(flags shared.Flags32) types.Set {
	values := []attr.Value{}
	for index, name := range []string{"des_crc", "des_md5", "rc4", "aes128", "aes256"} {
		if flags&(1<<index) != 0 {
			values = append(values, types.StringValue(name))
		}
	}
	return types.SetValueMust(types.StringType, values)
}

// mapState applies same-GUID observations only and preserves previously managed values when attributes are unavailable.
func mapState(model *ComputerResourceModel, observed *computers.Computer) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if observed == nil || observed.GUID == nil || *observed.GUID == (shared.GUID{}) || observed.Name == nil {
		diagnostics.AddError("Incomplete computer identity", "The SDK did not observe a usable GUID and name. Known state is retained.")
		return diagnostics
	}
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != observed.GUID.String() {
		diagnostics.AddError("Unexpected computer identity", "The SDK returned another GUID. Existing identity and state are retained.")
		return diagnostics
	}
	parent, valid := identity.ParentDN(observed.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete computer location", "A valid directory-observed distinguished name and parent were not returned.")
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
	properties := computers.ComputerProperties{}
	for _, field := range textProperties(model, &properties) {
		value := lifecycle.ObserveString(observed.Raw, field.ldap)
		diagnostics.Append(value.Diagnostics...)
		switch {
		case value.Usable:
			*field.value = value.Value
		case !field.value.IsNull() && !field.value.IsUnknown():
			diagnostics.AddWarning(
				"Computer property unavailable",
				"A managed property is hidden, absent or incomplete. Its prior value is retained; omission does not prove a clear.",
			)
		default:
			*field.value = types.StringNull()
		}
	}
	control := controlObservation(observed.AccountControl)
	for _, field := range boolProperties(model, &control) {
		if *field.target != nil {
			*field.value = types.BoolValue(**field.target)
		} else if field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	if observed.KerberosEncryptionTypes != nil {
		model.KerberosEncryptionTypes = observedEncryptionTypes(*observed.KerberosEncryptionTypes)
	}
	if observed.CompoundIdentitySupported != nil {
		model.CompoundIdentitySupported = types.BoolValue(*observed.CompoundIdentitySupported)
	}
	if observed.AccountExpires != nil {
		if *observed.AccountExpires > math.MaxInt64 {
			diagnostics.AddError("Unsupported computer expiration", "accountExpires exceeded the supported nonnegative FILETIME range. Prior state is retained.")
		} else {
			model.AccountExpires = types.Int64Value(int64(*observed.AccountExpires))
		}
	}
	return diagnostics
}

// persistState nulls the write-only input before serialization, retaining only acknowledged password-version metadata and stable identity.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model *ComputerResourceModel) diag.Diagnostics {
	model.PasswordWO = types.StringNull()
	properties, control := computers.ComputerProperties{}, accounts.AccountControlPatch{}
	for _, field := range textProperties(model, &properties) {
		if field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	for _, field := range boolProperties(model, &control) {
		if field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	for _, field := range []*types.String{&model.ID, &model.Name, &model.ParentDN, &model.DistinguishedName, &model.SID, &model.RecoveryDN} {
		if field.IsUnknown() {
			*field = types.StringNull()
		}
	}
	if model.KerberosEncryptionTypes.IsUnknown() {
		model.KerberosEncryptionTypes = types.SetNull(types.StringType)
	}
	if model.CompoundIdentitySupported.IsUnknown() {
		model.CompoundIdentitySupported = types.BoolNull()
	}
	if model.AccountExpires.IsUnknown() {
		model.AccountExpires = types.Int64Null()
	}
	if model.PasswordWOVersion.IsNull() || model.PasswordWOVersion.IsUnknown() {
		model.PasswordWOVersion = types.Int64Value(0)
	}
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() {
		canonical, err := identity.CanonicalGUID(model.ID.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid computer state identity", "The stored identity is not a nonzero GUID.")
		} else {
			diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
	return diagnostics
}
