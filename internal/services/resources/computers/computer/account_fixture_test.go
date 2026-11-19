// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// fixtureControl simulates directory observations, including the server's pwdLastSet change after password reset.
func fixtureControl(control accounts.AccountControlPatch, logon types.Bool) accounts.AccountControlState {
	state := accounts.AccountControlState{
		Enabled: control.Enabled, AccountNotDelegated: control.AccountNotDelegated, AllowReversiblePasswordEncryption: control.AllowReversiblePasswordEncryption,
		DoesNotRequirePreAuth: control.DoesNotRequirePreAuth, HomeDirectoryRequired: control.HomeDirectoryRequired, MNSLogonAccount: control.MNSLogonAccount,
		PasswordNeverExpires: control.PasswordNeverExpires, PasswordNotRequired: control.PasswordNotRequired, SmartcardLogonRequired: control.SmartcardLogonRequired,
		TrustedForDelegation: control.TrustedForDelegation, TrustedToAuthForDelegation: control.TrustedToAuthForDelegation, UseDESKeyOnly: control.UseDESKeyOnly,
	}
	if !logon.IsNull() && !logon.IsUnknown() {
		stamp := shared.FileTime(133000000000000000)
		if logon.ValueBool() {
			stamp = 0
		}
		state.PasswordLastSet = &stamp
	}
	return state
}
