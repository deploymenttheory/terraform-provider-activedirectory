// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import "github.com/hashicorp/terraform-plugin-framework/types"

// ServiceAccountResourceModel owns selected directory attributes and explicit password intent, never machine join or host configuration.
type ServiceAccountResourceModel struct {
	Kind                              types.String `tfsdk:"kind"`
	OutboundAuthenticationOnly        types.Bool   `tfsdk:"outbound_authentication_only"`
	ManagedPasswordIntervalDays       types.Int64  `tfsdk:"managed_password_interval_days"`
	ServicePrincipalNames             types.Set    `tfsdk:"service_principal_names"`
	PasswordReaderSIDs                types.Set    `tfsdk:"password_reader_sids"`
	ID                                types.String `tfsdk:"id"`
	Name                              types.String `tfsdk:"name"`
	ParentDN                          types.String `tfsdk:"parent_dn"`
	SAMAccountName                    types.String `tfsdk:"sam_account_name"`
	DNSHostName                       types.String `tfsdk:"dns_host_name"`
	DisplayName                       types.String `tfsdk:"display_name"`
	Description                       types.String `tfsdk:"description"`
	HomePage                          types.String `tfsdk:"home_page"`
	AccountExpires                    types.Int64  `tfsdk:"account_expires"`
	KerberosEncryptionTypes           types.Set    `tfsdk:"kerberos_encryption_types"`
	CompoundIdentitySupported         types.Bool   `tfsdk:"compound_identity_supported"`
	Enabled                           types.Bool   `tfsdk:"enabled"`
	AccountNotDelegated               types.Bool   `tfsdk:"account_not_delegated"`
	AllowReversiblePasswordEncryption types.Bool   `tfsdk:"allow_reversible_password_encryption"`
	DoesNotRequirePreAuth             types.Bool   `tfsdk:"does_not_require_pre_auth"`
	HomeDirectoryRequired             types.Bool   `tfsdk:"home_directory_required"`
	MNSLogonAccount                   types.Bool   `tfsdk:"mns_logon_account"`
	SmartcardLogonRequired            types.Bool   `tfsdk:"smartcard_logon_required"`
	TrustedForDelegation              types.Bool   `tfsdk:"trusted_for_delegation"`
	TrustedToAuthForDelegation        types.Bool   `tfsdk:"trusted_to_auth_for_delegation"`
	UseDESKeyOnly                     types.Bool   `tfsdk:"use_des_key_only"`
	PasswordWO                        types.String `tfsdk:"password_wo"`
	PasswordWOVersion                 types.Int64  `tfsdk:"password_wo_version"`
	DistinguishedName                 types.String `tfsdk:"distinguished_name"`
	SID                               types.String `tfsdk:"sid"`
	RecoveryDN                        types.String `tfsdk:"recovery_dn"`
	ClearProperties                   types.Set    `tfsdk:"clear_properties"`
}

// String withholds secret-bearing models from ordinary formatted diagnostics.
func (ServiceAccountResourceModel) String() string { return "ServiceAccountResourceModel{redacted}" }

// GoString also withholds the write-only value from Go-syntax debug formatting.
func (ServiceAccountResourceModel) GoString() string { return "ServiceAccountResourceModel{redacted}" }
