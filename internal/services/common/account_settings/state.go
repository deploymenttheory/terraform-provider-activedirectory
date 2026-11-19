// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package account_settings projects existing account state without creating accounts or exposing raw control masks.
package account_settings

import (
	"math"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// FlagValues keeps unavailable SDK fields null, distinct from a visible false flag.
func FlagValues(control accounts.AccountControlState) map[string]*bool {
	return map[string]*bool{
		"enabled":                              control.Enabled,
		"account_not_delegated":                control.AccountNotDelegated,
		"allow_reversible_password_encryption": control.AllowReversiblePasswordEncryption,
		"does_not_require_preauth":             control.DoesNotRequirePreAuth,
		"home_directory_required":              control.HomeDirectoryRequired,
		"mns_logon_account":                    control.MNSLogonAccount,
		"password_never_expires":               control.PasswordNeverExpires,
		"password_not_required":                control.PasswordNotRequired,
		"smartcard_logon_required":             control.SmartcardLogonRequired,
		"trusted_for_delegation":               control.TrustedForDelegation,
		"trusted_to_auth_for_delegation":       control.TrustedToAuthForDelegation,
		"use_des_key_only":                     control.UseDESKeyOnly,
	}
}

// AttributeTypes includes typed settings and server-computed observations; it never publishes a writable raw mask.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{
		"id": types.StringType, "distinguished_name": types.StringType, "expires_at": types.StringType, "never_expires": types.BoolType,
		"account_expires": types.Int64Type, "password_last_set": types.Int64Type,
		"change_password_at_logon": types.BoolType, "locked_out": types.BoolType, "password_expired": types.BoolType,
	}
	for name := range FlagValues(accounts.AccountControlState{}) {
		result[name] = types.BoolType
	}
	return result
}

// Project joins account-control and expiration reads only when both positively identify the same existing GUID.
func Project(control accounts.AccountControlState, expiration accounts.AccountExpiration) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	controlID, controlErr := control.Raw.Values("objectGUID").Single()
	expirationID, expirationErr := expiration.Raw.Values("objectGUID").Single()
	guid, guidErr := controlID.GUID()
	expiresGUID, expiresErr := expirationID.GUID()
	if controlErr != nil || expirationErr != nil || guidErr != nil || expiresErr != nil || guid == (shared.GUID{}) || expiresGUID != guid {
		diagnostics.AddError("Account settings identity is incomplete", "Both observations must identify the same nonzero account GUID. No combined state was emitted.")
		return nil, diagnostics
	}
	if _, err := shared.IdentityByDN(control.DistinguishedName); err != nil {
		diagnostics.AddError("Account settings location is incomplete", "A valid directory-observed distinguished name is required.")
		return nil, diagnostics
	}
	values := map[string]attr.Value{
		"id": types.StringValue(guid.String()), "distinguished_name": types.StringValue(control.DistinguishedName),
		"account_expires": types.Int64Null(), "password_last_set": types.Int64Null(), "change_password_at_logon": types.BoolNull(),
		"expires_at": types.StringNull(), "never_expires": types.BoolPointerValue(expiration.NeverExpires),
		"locked_out": types.BoolPointerValue(control.LockedOut), "password_expired": types.BoolPointerValue(control.PasswordExpired),
	}
	if expiration.ExpiresAt != nil {
		values["expires_at"] = types.StringValue(expiration.ExpiresAt.UTC().Format(time.RFC3339Nano))
	}
	for name, value := range FlagValues(control) {
		values[name] = types.BoolPointerValue(value)
	}
	for _, field := range []struct {
		name  string
		value *shared.FileTime
	}{
		{"account_expires", expiration.Value}, {"password_last_set", control.PasswordLastSet},
	} {
		if field.value == nil {
			continue
		}
		if *field.value > math.MaxInt64 {
			diagnostics.AddError("Account timestamp is outside the supported range", "Observed timestamps must be nonnegative signed FILETIME values. No state was emitted.")
			return nil, diagnostics
		}
		values[field.name] = types.Int64Value(int64(*field.value))
	}
	if control.PasswordLastSet != nil {
		values["change_password_at_logon"] = types.BoolValue(*control.PasswordLastSet == 0)
	}
	return values, diagnostics
}
