// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import "github.com/hashicorp/terraform-plugin-framework/types"

// AccountSettingsResourceModel separates an existing account's identity, explicitly owned settings and read-only computed state.
type AccountSettingsResourceModel struct {
	ID                                types.String `tfsdk:"id"`
	AccountGUID                       types.String `tfsdk:"account_guid"`
	DistinguishedName                 types.String `tfsdk:"distinguished_name"`
	Enabled                           types.Bool   `tfsdk:"enabled"`
	AccountNotDelegated               types.Bool   `tfsdk:"account_not_delegated"`
	AllowReversiblePasswordEncryption types.Bool   `tfsdk:"allow_reversible_password_encryption"`
	DoesNotRequirePreAuth             types.Bool   `tfsdk:"does_not_require_preauth"`
	HomeDirectoryRequired             types.Bool   `tfsdk:"home_directory_required"`
	MNSLogonAccount                   types.Bool   `tfsdk:"mns_logon_account"`
	PasswordNeverExpires              types.Bool   `tfsdk:"password_never_expires"`
	PasswordNotRequired               types.Bool   `tfsdk:"password_not_required"`
	SmartcardLogonRequired            types.Bool   `tfsdk:"smartcard_logon_required"`
	TrustedForDelegation              types.Bool   `tfsdk:"trusted_for_delegation"`
	TrustedToAuthForDelegation        types.Bool   `tfsdk:"trusted_to_auth_for_delegation"`
	UseDESKeyOnly                     types.Bool   `tfsdk:"use_des_key_only"`
	ChangePasswordAtLogon             types.Bool   `tfsdk:"change_password_at_logon"`
	ExpiresAt                         types.String `tfsdk:"expires_at"`
	NeverExpires                      types.Bool   `tfsdk:"never_expires"`
	AccountExpires                    types.Int64  `tfsdk:"account_expires"`
	PasswordLastSet                   types.Int64  `tfsdk:"password_last_set"`
	LockedOut                         types.Bool   `tfsdk:"locked_out"`
	PasswordExpired                   types.Bool   `tfsdk:"password_expired"`
}
