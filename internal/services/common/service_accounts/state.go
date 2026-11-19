// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package service_accounts projects managed account metadata without retrieving passwords or claiming host installation.
package service_accounts

import (
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// TextAttributes identifies the nonsecret scalar observations shared by managed and discovery surfaces.
func TextAttributes() map[string]string {
	return map[string]string{
		"sam_account_name": "sAMAccountName", "dns_host_name": "dNSHostName", "display_name": "displayName",
		"description": "description", "home_page": "wWWHomePage",
	}
}

// AttributeTypes excludes managed passwords, raw security descriptors, host installation and independently owned associations.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{
		"account_expires": types.Int64Type, "managed_password_interval_days": types.Int64Type,
		"kerberos_encryption_types": types.SetType{ElemType: types.StringType}, "service_principal_names": types.SetType{ElemType: types.StringType},
		"password_reader_sids": types.SetType{ElemType: types.StringType},
	}
	for _, name := range []string{"id", "name", "parent_dn", "distinguished_name", "sid", "kind"} {
		result[name] = types.StringType
	}
	for name := range TextAttributes() {
		result[name] = types.StringType
	}
	for _, name := range []string{
		"enabled", "account_not_delegated", "allow_reversible_password_encryption", "does_not_require_pre_auth",
		"home_directory_required", "mns_logon_account", "smartcard_logon_required", "trusted_for_delegation",
		"trusted_to_auth_for_delegation", "use_des_key_only", "compound_identity_supported",
	} {
		result[name] = types.BoolType
	}
	return result
}

// Project retains unavailable metadata as null and refuses classes outside the SDK's supported sMSA/gMSA lifecycle.
func Project(account sdkaccounts.ServiceAccount) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	kind := ""
	switch account.Kind {
	case sdkaccounts.ServiceAccountStandalone:
		kind = "standalone"
	case sdkaccounts.ServiceAccountGroupManaged:
		kind = "group_managed"
	}
	if account.GUID == nil || *account.GUID == (shared.GUID{}) || account.Name == nil || kind == "" {
		diagnostics.AddError("Incomplete service account identity", "A supported account kind, GUID and name are required. No import identity is emitted.")
		return nil, diagnostics
	}
	parent, valid := identity.ParentDN(account.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete service account location", "A valid distinguished name and parent are required.")
		return nil, diagnostics
	}
	values := map[string]attr.Value{
		"id": types.StringValue(account.GUID.String()), "name": types.StringValue(*account.Name), "parent_dn": types.StringValue(parent),
		"distinguished_name": types.StringValue(account.DistinguishedName), "sid": types.StringNull(), "kind": types.StringValue(kind),
		"account_expires": types.Int64Null(), "managed_password_interval_days": types.Int64Null(),
		"kerberos_encryption_types": types.SetNull(types.StringType), "password_reader_sids": types.SetNull(types.StringType),
	}
	if account.SID != nil {
		values["sid"] = types.StringValue(account.SID.String())
	}
	for field, ldap := range TextAttributes() {
		observed := lifecycle.ObserveString(account.Raw, ldap)
		diagnostics.Append(observed.Diagnostics...)
		values[field] = observed.Value
	}
	control := account.AccountControl
	for field, value := range map[string]*bool{
		"enabled": control.Enabled, "account_not_delegated": control.AccountNotDelegated, "allow_reversible_password_encryption": control.AllowReversiblePasswordEncryption,
		"does_not_require_pre_auth": control.DoesNotRequirePreAuth, "home_directory_required": control.HomeDirectoryRequired, "mns_logon_account": control.MNSLogonAccount,
		"smartcard_logon_required": control.SmartcardLogonRequired, "trusted_for_delegation": control.TrustedForDelegation,
		"trusted_to_auth_for_delegation": control.TrustedToAuthForDelegation, "use_des_key_only": control.UseDESKeyOnly,
		"compound_identity_supported": account.CompoundIdentitySupported,
	} {
		values[field] = types.BoolPointerValue(value)
	}
	if account.SupportedEncryptionTypes != nil {
		selected := []attr.Value{}
		for index, name := range []string{"des_crc", "des_md5", "rc4", "aes128", "aes256"} {
			if *account.SupportedEncryptionTypes&(1<<index) != 0 {
				selected = append(selected, types.StringValue(name))
			}
		}
		values["kerberos_encryption_types"] = types.SetValueMust(types.StringType, selected)
	}
	if account.AccountExpires != nil {
		if *account.AccountExpires > math.MaxInt64 {
			diagnostics.AddError("Unsupported service account expiration", "The observed expiration exceeds the supported nonnegative FILETIME range.")
		} else {
			values["account_expires"] = types.Int64Value(int64(*account.AccountExpires))
		}
	}
	if account.ManagedPasswordIntervalDays != nil {
		values["managed_password_interval_days"] = types.Int64Value(int64(*account.ManagedPasswordIntervalDays))
	}
	values["service_principal_names"] = types.SetNull(types.StringType)
	if account.ServicePrincipalNames.State == shared.AttributePresent {
		selected := []attr.Value{}
		for _, raw := range account.ServicePrincipalNames.Values {
			value, err := raw.Text()
			if err != nil {
				diagnostics.AddError("Invalid service principal observation", "The SDK returned a non-text service principal name.")
				return values, diagnostics
			}
			selected = append(selected, types.StringValue(value))
		}
		values["service_principal_names"] = types.SetValueMust(types.StringType, selected)
	}
	if account.PasswordReaderDescriptorReturned {
		selected := []attr.Value{}
		for _, sid := range account.PasswordReaders {
			selected = append(selected, types.StringValue(sid.String()))
		}
		values["password_reader_sids"] = types.SetValueMust(types.StringType, selected)
	}
	return values, diagnostics
}
