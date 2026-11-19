// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type textProperty struct {
	ldap   string
	value  *types.String
	target **string
}

// textProperties explicitly maps directory text without claiming that OS metadata configures the corresponding host.
func textProperties(model *ServiceAccountResourceModel, properties *service_accounts.ServiceAccountProperties) []textProperty {
	return []textProperty{
		{"sAMAccountName", &model.SAMAccountName, &properties.SAMAccountName},
		{"dNSHostName", &model.DNSHostName, &properties.DNSHostName},
		{"displayName", &model.DisplayName, &properties.DisplayName},
		{"description", &model.Description, &properties.Description},
		{"wWWHomePage", &model.HomePage, &properties.HomePage},
	}
}

type boolProperty struct {
	name   string
	value  *types.Bool
	target **bool
}

// boolProperties selects individual stored account settings without accepting a raw UAC or account-type replacement mask.
func boolProperties(model *ServiceAccountResourceModel, control *accounts.AccountControlPatch) []boolProperty {
	return []boolProperty{
		{"enabled", &model.Enabled, &control.Enabled},
		{"account_not_delegated", &model.AccountNotDelegated, &control.AccountNotDelegated},
		{"allow_reversible_password_encryption", &model.AllowReversiblePasswordEncryption, &control.AllowReversiblePasswordEncryption},
		{"does_not_require_pre_auth", &model.DoesNotRequirePreAuth, &control.DoesNotRequirePreAuth},
		{"home_directory_required", &model.HomeDirectoryRequired, &control.HomeDirectoryRequired},
		{"mns_logon_account", &model.MNSLogonAccount, &control.MNSLogonAccount},
		{"smartcard_logon_required", &model.SmartcardLogonRequired, &control.SmartcardLogonRequired},
		{"trusted_for_delegation", &model.TrustedForDelegation, &control.TrustedForDelegation},
		{"trusted_to_auth_for_delegation", &model.TrustedToAuthForDelegation, &control.TrustedToAuthForDelegation},
		{"use_des_key_only", &model.UseDESKeyOnly, &control.UseDESKeyOnly},
	}
}

// constructProperties preserves omission and literal empty text, exact expiration sentinels and independently owned encryption bits.
func constructProperties(model *ServiceAccountResourceModel) (service_accounts.ServiceAccountProperties, accounts.AccountControlPatch, diag.Diagnostics) {
	var properties service_accounts.ServiceAccountProperties
	var control accounts.AccountControlPatch
	var diagnostics diag.Diagnostics
	for _, field := range textProperties(model, &properties) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown service account property", "Resolve configured service account text properties before a mutation.")
		} else if !field.value.IsNull() {
			value := field.value.ValueString()
			if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
				diagnostics.AddError("Invalid service account text", "Service account properties must be valid UTF-8 directory text without NUL bytes.")
			} else {
				*field.target = &value
			}
		}
	}
	if !model.SAMAccountName.IsUnknown() && !model.SAMAccountName.IsNull() {
		sam := model.SAMAccountName.ValueString()
		if sam == "$" || !strings.HasSuffix(sam, "$") || strings.HasSuffix(sam, "$$") {
			diagnostics.AddError(
				"Canonical service account account name required",
				"sam_account_name must explicitly include one trailing $. This avoids server normalization changing a configured Terraform value.",
			)
		}
	}

	for _, field := range boolProperties(model, &control) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown service account account setting", "Resolve configured account settings before a mutation.")
		} else if !field.value.IsNull() {
			value := field.value.ValueBool()
			*field.target = &value
		}
	}

	if model.AccountExpires.IsUnknown() || !model.AccountExpires.IsNull() && model.AccountExpires.ValueInt64() < 0 {
		diagnostics.AddError(
			"Invalid service account expiration",
			"account_expires must be a known nonnegative FILETIME, including zero and 9223372036854775807 never-expire sentinels.",
		)
	} else if !model.AccountExpires.IsNull() {
		value, err := shared.ParseFileTime(strconv.FormatInt(model.AccountExpires.ValueInt64(), 10))
		if err != nil {
			diagnostics.AddError("Invalid service account expiration", "The SDK rejected the requested FILETIME.")
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
	spns, spnDiagnostics := stringValues(model.ServicePrincipalNames)
	diagnostics.Append(spnDiagnostics...)
	properties.ServicePrincipalNames = spns
	return properties, control, diagnostics
}

// encryptionTypes selects only the SDK-supported low five bits; the SDK preserves compound identity and unrelated returned bits.
func encryptionTypes(value types.Set) (*shared.Flags32, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown service account encryption intent", "Resolve kerberos_encryption_types before a mutation.")
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
			diagnostics.AddError(
				"Invalid service account encryption type",
				"Use explicit des_crc, des_md5, rc4, aes128 or aes256 names. No raw mask or implicit fallback is accepted.",
			)
		} else {
			flags |= allowed[text.ValueString()]
		}
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return &flags, diagnostics
}

// stringValues distinguishes omission from an explicitly empty complete set and validates every value before a mutation phase.
func stringValues(value types.Set) (*[]string, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown service account collection", "Resolve all configured collection values before any mutation.")
		return nil, diagnostics
	}
	if value.IsNull() {
		return nil, diagnostics
	}
	values := []string{}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsUnknown() || text.IsNull() || text.ValueString() == "" || !utf8.ValidString(text.ValueString()) || strings.ContainsRune(text.ValueString(), 0) {
			diagnostics.AddError("Invalid service account collection value", "Collection values must be known nonempty directory text without NUL bytes.")
			continue
		}
		values = append(values, text.ValueString())
	}
	return &values, diagnostics
}

// constructReaders accepts canonical SIDs only; the SDK resolves each principal and preserves unrelated descriptor components.
func constructReaders(value types.Set) (*[]shared.Identity, diag.Diagnostics) {
	values, diagnostics := stringValues(value)
	if values == nil {
		return nil, diagnostics
	}
	readers := []shared.Identity{}
	for _, value := range *values {
		sid, err := shared.ParseSID(value)
		if err != nil || sid.String() != value {
			diagnostics.AddError("Invalid password reader SID", "password_reader_sids requires canonical principal SIDs; no name or DN is inferred.")
			continue
		}
		principal, err := shared.IdentityBySID(value)
		if err != nil {
			diagnostics.AddError("Invalid password reader identity", "The SDK rejected the selected principal SID.")
			continue
		}
		readers = append(readers, principal)
	}
	return &readers, diagnostics
}

// validateNames checks all location and typed-property intent before an earlier rename or move may have accepted effects.
func validateNames(model *ServiceAccountResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if model.Name.IsNull() || model.Name.IsUnknown() || strings.TrimSpace(model.Name.ValueString()) == "" ||
		!utf8.ValidString(model.Name.ValueString()) || strings.ContainsRune(model.Name.ValueString(), 0) {
		diagnostics.AddError("Invalid service account name", "A known nonempty literal name without NUL is required.")
	}
	if model.SAMAccountName.IsNull() || model.SAMAccountName.IsUnknown() || model.SAMAccountName.ValueString() == "" {
		diagnostics.AddError("Explicit service account name required", "Provide a known canonical sam_account_name.")
	}
	if model.ParentDN.IsNull() || model.ParentDN.IsUnknown() {
		diagnostics.AddError("Explicit service account parent required", "Select a known parent_dn; no container is inferred.")
	} else if _, err := shared.IdentityByDN(model.ParentDN.ValueString()); err != nil {
		diagnostics.AddError("Invalid service account parent", "parent_dn must be an explicit distinguished name.")
	}
	if model.Kind.IsUnknown() || model.Kind.IsNull() || (model.Kind.ValueString() != "standalone" && model.Kind.ValueString() != "group_managed") {
		diagnostics.AddError("Invalid service account kind", "Select standalone or group_managed. Ordinary dMSA creation is unavailable.")
	}
	_, readerDiagnostics := constructReaders(model.PasswordReaderSIDs)
	diagnostics.Append(readerDiagnostics...)
	if model.Kind.ValueString() == "standalone" && !model.PasswordReaderSIDs.IsNull() {
		diagnostics.AddError("Standalone password readers unavailable", "Reader-policy grants belong to group-managed service accounts.")
	}
	return diagnostics
}

// constructCreate builds one SDK staged creation request and never asks the directory to return a managed password.
func constructCreate(model *ServiceAccountResourceModel, password types.String) (service_accounts.CreateServiceAccountRequest, diag.Diagnostics) {
	properties, control, diagnostics := constructProperties(model)
	diagnostics.Append(validateNames(model)...)
	request := service_accounts.CreateServiceAccountRequest{Name: model.Name.ValueString(), Properties: properties, Control: control}
	request.Parent, _ = shared.IdentityByDN(model.ParentDN.ValueString())
	request.Kind = service_accounts.ServiceAccountStandalone
	if model.Kind.ValueString() == "group_managed" {
		request.Kind = service_accounts.ServiceAccountGroupManaged
	}
	request.OutboundAuthenticationOnly = model.OutboundAuthenticationOnly.ValueBool()
	if model.OutboundAuthenticationOnly.IsUnknown() {
		diagnostics.AddError("Unknown creation form", "Resolve outbound_authentication_only before creation.")
	}
	if model.ManagedPasswordIntervalDays.IsUnknown() {
		diagnostics.AddError("Unknown managed password interval", "Resolve the configured interval before creation.")
	} else if !model.ManagedPasswordIntervalDays.IsNull() {
		days := model.ManagedPasswordIntervalDays.ValueInt64()
		if days < 1 || days > 2147483647 || request.Kind != service_accounts.ServiceAccountGroupManaged {
			diagnostics.AddError("Invalid managed password interval", "An explicit interval from 1 to 2147483647 days is available only for gMSA creation.")
		} else {
			value := uint32(days)
			request.ManagedPasswordIntervalDays = &value
		}
	}
	readers, readerDiagnostics := constructReaders(model.PasswordReaderSIDs)
	diagnostics.Append(readerDiagnostics...)
	if readers != nil {
		request.PasswordReaders = *readers
	}
	if request.OutboundAuthenticationOnly && (request.Kind != service_accounts.ServiceAccountGroupManaged ||
		properties.DNSHostName != nil || request.ManagedPasswordIntervalDays != nil || readers != nil || properties.CompoundIdentitySupported != nil) {
		diagnostics.AddError("Conflicting outbound creation inputs", "Outbound-only gMSA creation excludes DNS host name, interval, reader grants and compound identity inputs.")
	}
	if request.Kind == service_accounts.ServiceAccountGroupManaged && !request.OutboundAuthenticationOnly &&
		(properties.DNSHostName == nil || *properties.DNSHostName == "") {
		diagnostics.AddError("Group-managed DNS name required", "Provide dns_host_name for the ordinary gMSA creation form.")
	}
	diagnostics.Append(validateCreationPassword(model, password, request.Kind)...)
	if model.ClearProperties.IsUnknown() || len(model.ClearProperties.Elements()) != 0 {
		diagnostics.AddError("Service account clear requires an existing object", "Explicit property clearing applies only to existing or imported accounts.")
	}
	if diagnostics.HasError() {
		return service_accounts.CreateServiceAccountRequest{}, diagnostics
	}
	if !password.IsNull() {
		request.InitialPassword = []byte(password.ValueString())
	}
	return request, diagnostics
}

// validateCreationPassword separates sMSA creation input from DC-managed gMSA password material.
func validateCreationPassword(model *ServiceAccountResourceModel, password types.String, kind service_accounts.ServiceAccountKind) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	version := model.PasswordWOVersion.ValueInt64()
	if model.PasswordWOVersion.IsUnknown() || version < 0 || password.IsUnknown() ||
		password.IsNull() && version != 0 || !password.IsNull() && (version < 1 || password.ValueString() == "" ||
		!utf8.ValidString(password.ValueString()) || strings.ContainsRune(password.ValueString(), 0)) {
		diagnostics.AddError(
			"Invalid creation password intent",
			"A known nonempty sMSA password_wo requires a positive password_wo_version; omitted passwords require version zero.",
		)
	}
	if kind != service_accounts.ServiceAccountStandalone && (!password.IsNull() || version != 0) {
		diagnostics.AddError("Group-managed password input refused", "The DC manages gMSA passwords. No explicit password input is accepted or read back.")
	}
	return diagnostics
}
