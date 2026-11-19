// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"math"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

type flagProperty struct {
	name   string
	value  *types.Bool
	target **bool
}

// flagProperties is the complete typed write mapping; computed lockout and password-expiry flags cannot enter it.
func flagProperties(model *AccountSettingsResourceModel, patch *accounts.AccountControlPatch) []flagProperty {
	return []flagProperty{
		{"enabled", &model.Enabled, &patch.Enabled},
		{"account_not_delegated", &model.AccountNotDelegated, &patch.AccountNotDelegated},
		{"allow_reversible_password_encryption", &model.AllowReversiblePasswordEncryption, &patch.AllowReversiblePasswordEncryption},
		{"does_not_require_preauth", &model.DoesNotRequirePreAuth, &patch.DoesNotRequirePreAuth},
		{"home_directory_required", &model.HomeDirectoryRequired, &patch.HomeDirectoryRequired},
		{"mns_logon_account", &model.MNSLogonAccount, &patch.MNSLogonAccount},
		{"password_never_expires", &model.PasswordNeverExpires, &patch.PasswordNeverExpires},
		{"password_not_required", &model.PasswordNotRequired, &patch.PasswordNotRequired},
		{"smartcard_logon_required", &model.SmartcardLogonRequired, &patch.SmartcardLogonRequired},
		{"trusted_for_delegation", &model.TrustedForDelegation, &patch.TrustedForDelegation},
		{"trusted_to_auth_for_delegation", &model.TrustedToAuthForDelegation, &patch.TrustedToAuthForDelegation},
		{"use_des_key_only", &model.UseDESKeyOnly, &patch.UseDESKeyOnly},
		{"change_password_at_logon", &model.ChangePasswordAtLogon, &patch.ChangePasswordAtLogon},
	}
}

type settingsIntent struct {
	GUID            string
	Patch           accounts.AccountControlPatch
	Flags           map[string]bool
	ExpiresAt       *time.Time
	ClearExpiration bool
}

// constructSettings validates every configured phase before any SDK effect and preserves null as non-ownership.
func constructSettings(config *AccountSettingsResourceModel) (settingsIntent, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	intent := settingsIntent{Flags: map[string]bool{}}
	guid, err := identity.CanonicalGUID(config.AccountGUID.ValueString())
	if config.AccountGUID.IsNull() || config.AccountGUID.IsUnknown() || err != nil {
		diagnostics.AddError("Invalid existing account identity", "account_guid must be a known nonzero object GUID. This resource never creates an account.")
	} else {
		intent.GUID = guid
	}
	for _, field := range flagProperties(config, &intent.Patch) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown account setting", field.name+" must be known before any account settings write.")
			continue
		}
		if field.value.IsNull() {
			continue
		}
		value := field.value.ValueBool()
		*field.target = &value
		intent.Flags[field.name] = value
	}
	switch {
	case config.ExpiresAt.IsUnknown() || config.NeverExpires.IsUnknown():
		diagnostics.AddError("Unknown account expiration intent", "Resolve expires_at and never_expires before any account-control or expiration write.")
	case !config.ExpiresAt.IsNull():
		at, parseErr := time.Parse(time.RFC3339Nano, config.ExpiresAt.ValueString())
		stamp, stampErr := shared.FileTimeFromTime(at)
		if parseErr != nil || stampErr != nil || stamp == 0 || stamp > math.MaxInt64 ||
			at.Nanosecond()%100 != 0 || at.UTC().Format(time.RFC3339Nano) != config.ExpiresAt.ValueString() {
			diagnostics.AddError(
				"Invalid account expiration instant",
				"expires_at must be an explicit canonical UTC RFC3339 instant at 100-nanosecond precision or coarser, within the directory range.",
			)
		} else {
			intent.ExpiresAt = &at
		}
		if !config.NeverExpires.IsNull() && config.NeverExpires.ValueBool() {
			diagnostics.AddError("Conflicting account expiration intent", "expires_at and never_expires = true cannot be configured together.")
		}
	case !config.NeverExpires.IsNull():
		if !config.NeverExpires.ValueBool() {
			diagnostics.AddError(
				"Finite account expiration requires an instant",
				"Set expires_at to select a finite expiration; never_expires = false cannot choose an expiration time.",
			)
		} else {
			intent.ClearExpiration = true
		}
	}
	if intent.Patch.ChangePasswordAtLogon != nil && *intent.Patch.ChangePasswordAtLogon &&
		intent.Patch.PasswordNeverExpires != nil && *intent.Patch.PasswordNeverExpires {
		diagnostics.AddError("Conflicting password settings", "change_password_at_logon = true cannot be combined with password_never_expires = true.")
	}
	return intent, diagnostics
}
