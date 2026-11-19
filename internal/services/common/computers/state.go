// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package computers projects typed directory observations without claiming host join status or exposing secrets.
package computers

import (
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkcomputers "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// TextAttributes maps selected single-valued directory text to public Terraform observations.
func TextAttributes() map[string]string {
	return map[string]string{
		"sam_account_name": "sAMAccountName", "dns_host_name": "dNSHostName", "user_principal_name": "userPrincipalName",
		"display_name": "displayName", "description": "description", "location": "location", "managed_by_dn": "managedBy", "home_page": "wWWHomePage",
		"operating_system": "operatingSystem", "operating_system_version": "operatingSystemVersion", "operating_system_service_pack": "operatingSystemServicePack",
		"operating_system_hotfix": "operatingSystemHotfix",
	}
}

// AttributeTypes excludes passwords, provider-local versions, raw masks and independently owned relationships.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{"account_expires": types.Int64Type, "kerberos_encryption_types": types.SetType{ElemType: types.StringType}}
	for _, name := range []string{"id", "name", "parent_dn", "distinguished_name", "sid"} {
		result[name] = types.StringType
	}
	for name := range TextAttributes() {
		result[name] = types.StringType
	}
	for _, name := range []string{
		"enabled", "account_not_delegated", "allow_reversible_password_encryption", "does_not_require_pre_auth",
		"home_directory_required", "mns_logon_account", "password_never_expires", "password_not_required", "smartcard_logon_required",
		"trusted_for_delegation", "trusted_to_auth_for_delegation", "use_des_key_only", "change_password_at_logon", "compound_identity_supported",
	} {
		result[name] = types.BoolType
	}
	return result
}

// Project requires usable GUID/location and preserves unavailable observations as null; directory metadata never implies machine configuration.
func Project(computer sdkcomputers.Computer) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if computer.GUID == nil || *computer.GUID == (shared.GUID{}) || computer.Name == nil {
		diagnostics.AddError("Incomplete computer identity", "A usable GUID and name are required. No import identity is emitted.")
		return nil, diagnostics
	}
	parent, valid := identity.ParentDN(computer.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete computer location", "A valid distinguished name and parent are required.")
		return nil, diagnostics
	}
	values := map[string]attr.Value{
		"id": types.StringValue(computer.GUID.String()), "name": types.StringValue(*computer.Name), "parent_dn": types.StringValue(parent),
		"distinguished_name": types.StringValue(computer.DistinguishedName), "sid": types.StringNull(), "account_expires": types.Int64Null(),
		"kerberos_encryption_types": types.SetNull(types.StringType), "change_password_at_logon": types.BoolNull(),
	}
	if computer.SID != nil {
		values["sid"] = types.StringValue(computer.SID.String())
	}
	for field, ldap := range TextAttributes() {
		observed := lifecycle.ObserveString(computer.Raw, ldap)
		diagnostics.Append(observed.Diagnostics...)
		values[field] = observed.Value
	}
	control := computer.AccountControl
	for field, value := range map[string]*bool{
		"enabled": control.Enabled, "account_not_delegated": control.AccountNotDelegated, "allow_reversible_password_encryption": control.AllowReversiblePasswordEncryption,
		"does_not_require_pre_auth": control.DoesNotRequirePreAuth, "home_directory_required": control.HomeDirectoryRequired, "mns_logon_account": control.MNSLogonAccount,
		"password_never_expires": control.PasswordNeverExpires, "password_not_required": control.PasswordNotRequired, "smartcard_logon_required": control.SmartcardLogonRequired,
		"trusted_for_delegation": control.TrustedForDelegation, "trusted_to_auth_for_delegation": control.TrustedToAuthForDelegation, "use_des_key_only": control.UseDESKeyOnly,
		"compound_identity_supported": computer.CompoundIdentitySupported,
	} {
		values[field] = types.BoolPointerValue(value)
	}
	if control.PasswordLastSet != nil {
		values["change_password_at_logon"] = types.BoolValue(*control.PasswordLastSet == 0)
	}
	if computer.KerberosEncryptionTypes != nil {
		selected := []attr.Value{}
		for index, name := range []string{"des_crc", "des_md5", "rc4", "aes128", "aes256"} {
			if *computer.KerberosEncryptionTypes&(1<<index) != 0 {
				selected = append(selected, types.StringValue(name))
			}
		}
		values["kerberos_encryption_types"] = types.SetValueMust(types.StringType, selected)
	}
	if computer.AccountExpires != nil {
		if *computer.AccountExpires > math.MaxInt64 {
			diagnostics.AddError("Unsupported computer expiration", "The observed expiration exceeds the supported nonnegative FILETIME range.")
		} else {
			values["account_expires"] = types.Int64Value(int64(*computer.AccountExpires))
		}
	}
	return values, diagnostics
}
