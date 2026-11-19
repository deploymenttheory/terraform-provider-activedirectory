// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type textProperty struct {
	ldap   string
	value  *types.String
	target **string
}

// textProperties explicitly maps directory text without claiming that OS metadata configures the corresponding host.
func textProperties(model *ComputerResourceModel, properties *computers.ComputerProperties) []textProperty {
	return []textProperty{
		{"sAMAccountName", &model.SAMAccountName, &properties.SAMAccountName},
		{"dNSHostName", &model.DNSHostName, &properties.DNSHostName},
		{"userPrincipalName", &model.UserPrincipalName, &properties.UserPrincipalName},
		{"displayName", &model.DisplayName, &properties.DisplayName},
		{"description", &model.Description, &properties.Description},
		{"location", &model.Location, &properties.Location},
		{"managedBy", &model.ManagedByDN, &properties.ManagedByDN},
		{"wWWHomePage", &model.HomePage, &properties.HomePage},
		{"operatingSystem", &model.OperatingSystem, &properties.OperatingSystem},
		{"operatingSystemVersion", &model.OperatingSystemVersion, &properties.OperatingSystemVersion},
		{"operatingSystemServicePack", &model.OperatingSystemServicePack, &properties.OperatingSystemServicePack},
		{"operatingSystemHotfix", &model.OperatingSystemHotfix, &properties.OperatingSystemHotfix},
	}
}

type boolProperty struct {
	name   string
	value  *types.Bool
	target **bool
}

// boolProperties selects individual stored account settings without accepting a raw UAC or account-type replacement mask.
func boolProperties(model *ComputerResourceModel, control *accounts.AccountControlPatch) []boolProperty {
	return []boolProperty{
		{"enabled", &model.Enabled, &control.Enabled},
		{"account_not_delegated", &model.AccountNotDelegated, &control.AccountNotDelegated},
		{"allow_reversible_password_encryption", &model.AllowReversiblePasswordEncryption, &control.AllowReversiblePasswordEncryption},
		{"does_not_require_pre_auth", &model.DoesNotRequirePreAuth, &control.DoesNotRequirePreAuth},
		{"home_directory_required", &model.HomeDirectoryRequired, &control.HomeDirectoryRequired},
		{"mns_logon_account", &model.MNSLogonAccount, &control.MNSLogonAccount},
		{"password_never_expires", &model.PasswordNeverExpires, &control.PasswordNeverExpires},
		{"password_not_required", &model.PasswordNotRequired, &control.PasswordNotRequired},
		{"smartcard_logon_required", &model.SmartcardLogonRequired, &control.SmartcardLogonRequired},
		{"trusted_for_delegation", &model.TrustedForDelegation, &control.TrustedForDelegation},
		{"trusted_to_auth_for_delegation", &model.TrustedToAuthForDelegation, &control.TrustedToAuthForDelegation},
		{"use_des_key_only", &model.UseDESKeyOnly, &control.UseDESKeyOnly},
		{"change_password_at_logon", &model.ChangePasswordAtLogon, &control.ChangePasswordAtLogon},
	}
}

// constructProperties preserves omission and literal empty text, exact expiration sentinels and independently owned encryption bits.
func constructProperties(model *ComputerResourceModel) (computers.ComputerProperties, accounts.AccountControlPatch, diag.Diagnostics) {
	var properties computers.ComputerProperties
	var control accounts.AccountControlPatch
	var diagnostics diag.Diagnostics
	for _, field := range textProperties(model, &properties) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown computer property", "Resolve configured computer text properties before a mutation.")
		} else if !field.value.IsNull() {
			value := field.value.ValueString()
			if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
				diagnostics.AddError("Invalid computer text", "Computer properties must be valid UTF-8 directory text without NUL bytes.")
			} else {
				*field.target = &value
			}
		}
	}
	if !model.SAMAccountName.IsUnknown() && !model.SAMAccountName.IsNull() {
		sam := model.SAMAccountName.ValueString()
		if sam == "$" || !strings.HasSuffix(sam, "$") || strings.HasSuffix(sam, "$$") {
			diagnostics.AddError(
				"Canonical computer account name required",
				"sam_account_name must explicitly include one trailing $. This avoids server normalization changing a configured Terraform value.",
			)
		}
	}
	if properties.ManagedByDN != nil {
		if _, err := shared.IdentityByDN(*properties.ManagedByDN); err != nil {
			diagnostics.AddError("Invalid computer manager", "managed_by_dn must be an explicit distinguished name; no principal lookup is inferred.")
		}
	}
	for _, field := range boolProperties(model, &control) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown computer account setting", "Resolve configured account settings before a mutation.")
		} else if !field.value.IsNull() {
			value := field.value.ValueBool()
			*field.target = &value
		}
	}
	if model.PasswordNeverExpires.ValueBool() && model.ChangePasswordAtLogon.ValueBool() {
		diagnostics.AddError("Conflicting computer password settings", "password_never_expires and change_password_at_logon cannot both be true.")
	}
	if model.AccountExpires.IsUnknown() || !model.AccountExpires.IsNull() && model.AccountExpires.ValueInt64() < 0 {
		diagnostics.AddError("Invalid computer expiration", "account_expires must be a known nonnegative FILETIME, including zero and 9223372036854775807 never-expire sentinels.")
	} else if !model.AccountExpires.IsNull() {
		value, err := shared.ParseFileTime(strconv.FormatInt(model.AccountExpires.ValueInt64(), 10))
		if err != nil {
			diagnostics.AddError("Invalid computer expiration", "The SDK rejected the requested FILETIME.")
		} else {
			properties.AccountExpires = &value
		}
	}
	if model.CompoundIdentitySupported.IsUnknown() {
		diagnostics.AddError("Unknown compound identity setting", "Resolve compound_identity_supported before a mutation.")
	} else if !model.CompoundIdentitySupported.IsNull() {
		value := model.CompoundIdentitySupported.ValueBool()
		properties.CompoundIdentitySupported = &value
	}
	flags, encryptionDiagnostics := encryptionTypes(model.KerberosEncryptionTypes)
	properties.KerberosEncryptionTypes = flags
	diagnostics.Append(encryptionDiagnostics...)
	return properties, control, diagnostics
}

// encryptionTypes selects only the SDK-supported low five bits; the SDK preserves compound identity and unrelated returned bits.
func encryptionTypes(value types.Set) (*shared.Flags32, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown computer encryption intent", "Resolve kerberos_encryption_types before a mutation.")
		return nil, diagnostics
	}
	if value.IsNull() {
		return nil, diagnostics
	}
	allowed := map[string]shared.Flags32{"des_crc": 1, "des_md5": 2, "rc4": 4, "aes128": 8, "aes256": 16}
	var flags shared.Flags32
	for _, element := range value.Elements() {
		text, known := element.(types.String)
		if !known || text.IsNull() || text.IsUnknown() || allowed[text.ValueString()] == 0 {
			diagnostics.AddError("Invalid computer encryption type", "Use explicit des_crc, des_md5, rc4, aes128 or aes256 names. No raw mask or implicit fallback is accepted.")
		} else {
			flags |= allowed[text.ValueString()]
		}
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return &flags, diagnostics
}

// constructCreate requests only directory creation with explicit identity and staged password/enablement; it never joins a machine.
func constructCreate(model *ComputerResourceModel, password types.String) (computers.CreateComputerRequest, diag.Diagnostics) {
	properties, control, diagnostics := constructProperties(model)
	request := computers.CreateComputerRequest{Name: model.Name.ValueString(), Properties: properties, Control: control}
	if model.Name.IsNull() || model.Name.IsUnknown() || strings.TrimSpace(request.Name) == "" || !utf8.ValidString(request.Name) || strings.ContainsRune(request.Name, 0) ||
		model.SAMAccountName.IsNull() || model.SAMAccountName.IsUnknown() || model.SAMAccountName.ValueString() == "" {
		diagnostics.AddError("Explicit computer names required", "Provide known nonempty name and canonical sam_account_name before directory creation.")
	}
	if model.ParentDN.IsNull() || model.ParentDN.IsUnknown() {
		diagnostics.AddError("Explicit computer parent required", "Select a known parent_dn; no default Computers container is inferred.")
	} else {
		var err error
		request.Parent, err = shared.IdentityByDN(model.ParentDN.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid computer parent", "parent_dn must be a distinguished name.")
		}
	}
	if model.PasswordWOVersion.IsUnknown() || model.PasswordWOVersion.ValueInt64() < 0 || password.IsUnknown() {
		diagnostics.AddError("Unknown computer password intent", "Password input and its nonnegative version must be known before creation.")
	}
	if password.IsNull() && model.PasswordWOVersion.ValueInt64() > 0 || !password.IsNull() && (model.PasswordWOVersion.ValueInt64() < 1 ||
		password.ValueString() == "" || !utf8.ValidString(password.ValueString()) || strings.ContainsRune(password.ValueString(), 0)) {
		diagnostics.AddError(
			"Invalid computer password intent",
			"A nonempty password_wo requires an explicit positive password_wo_version. Secret values are never recovered from state.",
		)
	}
	if model.Enabled.ValueBool() && password.IsNull() && !model.PasswordNotRequired.ValueBool() {
		diagnostics.AddError("Initial computer password required", "Enabling a new computer account requires password setup or explicit password_not_required=true.")
	}
	if model.ClearProperties.IsUnknown() || !model.ClearProperties.IsNull() && len(model.ClearProperties.Elements()) > 0 {
		diagnostics.AddError("Computer clear requires an existing object", "Explicit property clears apply only to an existing or imported computer account.")
	}
	if diagnostics.HasError() {
		return computers.CreateComputerRequest{}, diagnostics
	}
	if !password.IsNull() {
		request.InitialPassword = []byte(password.ValueString())
	}
	return request, diagnostics
}
