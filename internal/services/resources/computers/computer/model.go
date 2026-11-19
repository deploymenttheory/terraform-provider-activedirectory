// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import "github.com/hashicorp/terraform-plugin-framework/types"

// ComputerResourceModel owns selected directory attributes and explicit password intent, never machine join or host configuration.
type ComputerResourceModel struct {
	ID                                types.String `tfsdk:"id"`
	Name                              types.String `tfsdk:"name"`
	ParentDN                          types.String `tfsdk:"parent_dn"`
	SAMAccountName                    types.String `tfsdk:"sam_account_name"`
	DNSHostName                       types.String `tfsdk:"dns_host_name"`
	UserPrincipalName                 types.String `tfsdk:"user_principal_name"`
	DisplayName                       types.String `tfsdk:"display_name"`
	Description                       types.String `tfsdk:"description"`
	Location                          types.String `tfsdk:"location"`
	ManagedByDN                       types.String `tfsdk:"managed_by_dn"`
	HomePage                          types.String `tfsdk:"home_page"`
	OperatingSystem                   types.String `tfsdk:"operating_system"`
	OperatingSystemVersion            types.String `tfsdk:"operating_system_version"`
	OperatingSystemServicePack        types.String `tfsdk:"operating_system_service_pack"`
	OperatingSystemHotfix             types.String `tfsdk:"operating_system_hotfix"`
	AccountExpires                    types.Int64  `tfsdk:"account_expires"`
	KerberosEncryptionTypes           types.Set    `tfsdk:"kerberos_encryption_types"`
	CompoundIdentitySupported         types.Bool   `tfsdk:"compound_identity_supported"`
	Enabled                           types.Bool   `tfsdk:"enabled"`
	AccountNotDelegated               types.Bool   `tfsdk:"account_not_delegated"`
	AllowReversiblePasswordEncryption types.Bool   `tfsdk:"allow_reversible_password_encryption"`
	DoesNotRequirePreAuth             types.Bool   `tfsdk:"does_not_require_pre_auth"`
	HomeDirectoryRequired             types.Bool   `tfsdk:"home_directory_required"`
	MNSLogonAccount                   types.Bool   `tfsdk:"mns_logon_account"`
	PasswordNeverExpires              types.Bool   `tfsdk:"password_never_expires"`
	PasswordNotRequired               types.Bool   `tfsdk:"password_not_required"`
	SmartcardLogonRequired            types.Bool   `tfsdk:"smartcard_logon_required"`
	TrustedForDelegation              types.Bool   `tfsdk:"trusted_for_delegation"`
	TrustedToAuthForDelegation        types.Bool   `tfsdk:"trusted_to_auth_for_delegation"`
	UseDESKeyOnly                     types.Bool   `tfsdk:"use_des_key_only"`
	ChangePasswordAtLogon             types.Bool   `tfsdk:"change_password_at_logon"`
	PasswordWO                        types.String `tfsdk:"password_wo"`
	PasswordWOVersion                 types.Int64  `tfsdk:"password_wo_version"`
	DistinguishedName                 types.String `tfsdk:"distinguished_name"`
	SID                               types.String `tfsdk:"sid"`
	RecoveryDN                        types.String `tfsdk:"recovery_dn"`
	ClearProperties                   types.Set    `tfsdk:"clear_properties"`
}

// String withholds secret-bearing models from ordinary formatted diagnostics.
func (ComputerResourceModel) String() string { return "ComputerResourceModel{redacted}" }

// GoString also withholds the write-only value from Go-syntax debug formatting.
func (ComputerResourceModel) GoString() string { return "ComputerResourceModel{redacted}" }
