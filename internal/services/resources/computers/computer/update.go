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

// validateUpdate checks every later property/password phase before an earlier disable, rename or move can have accepted effects.
func validateUpdate(current, plan *ComputerResourceModel, password types.String) diag.Diagnostics {
	_, _, diagnostics := constructProperties(plan)
	if plan.Name.IsNull() || plan.Name.IsUnknown() || strings.TrimSpace(plan.Name.ValueString()) == "" || !utf8.ValidString(plan.Name.ValueString()) ||
		strings.ContainsRune(plan.Name.ValueString(), 0) {
		diagnostics.AddError("Invalid computer name", "A known, nonempty literal name without NUL is required before any phase.")
	}
	if plan.SAMAccountName.IsNull() || plan.SAMAccountName.IsUnknown() || plan.SAMAccountName.ValueString() == "" {
		diagnostics.AddError("Invalid computer account name", "Provide a known canonical sam_account_name before any phase.")
	}
	if plan.ParentDN.IsNull() || plan.ParentDN.IsUnknown() {
		diagnostics.AddError("Unknown computer parent", "Resolve the explicit parent_dn before any mutation.")
	} else if _, err := shared.IdentityByDN(plan.ParentDN.ValueString()); err != nil {
		diagnostics.AddError("Invalid computer parent", "parent_dn must be an explicit distinguished name.")
	}
	if plan.PasswordWOVersion.IsNull() || plan.PasswordWOVersion.IsUnknown() || plan.PasswordWOVersion.ValueInt64() < current.PasswordWOVersion.ValueInt64() {
		diagnostics.AddError("Invalid computer password version", "Retain or increase the acknowledged password_wo_version.")
	} else if plan.PasswordWOVersion.ValueInt64() > current.PasswordWOVersion.ValueInt64() {
		if password.IsNull() || password.IsUnknown() || password.ValueString() == "" || !utf8.ValidString(password.ValueString()) ||
			strings.ContainsRune(password.ValueString(), 0) {
			diagnostics.AddError("New computer password required", "An increased version requires known nonempty password_wo before any mutation phase.")
		}
	}
	_, clearDiagnostics := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	return diagnostics
}

// constructUpdate emits selected changed properties, explicit clears and semantic account/encryption patches; it never replaces a raw mask.
func constructUpdate(current, plan *ComputerResourceModel, target shared.Identity) (computers.UpdateComputerRequest, bool) {
	request := computers.UpdateComputerRequest{Identity: target}
	ignored := computers.ComputerProperties{}
	before, after := textProperties(current, &ignored), textProperties(plan, &request.Properties)
	previousClears, _ := clearProperties(current.ClearProperties)
	requestedClears, _ := clearProperties(plan.ClearProperties)
	changed := false
	for index, field := range after {
		selected := requestedClears[strings.ToLower(field.ldap)]
		newClear := selected && !previousClears[strings.ToLower(field.ldap)]
		if field.value.IsNull() && !selected || field.value.Equal(*before[index].value) && !newClear {
			continue
		}
		changed = true
		if field.value.IsNull() {
			request.Changes = append(request.Changes, shared.AttributeChange{Attribute: field.ldap, Kind: shared.AttributeClear})
		} else {
			value := field.value.ValueString()
			*field.target = &value
		}
	}
	control := accounts.AccountControlPatch{}
	oldFlags, newFlags := boolProperties(current, &control), boolProperties(plan, &request.Control)
	for index, field := range newFlags {
		if field.value.IsNull() || field.value.Equal(*oldFlags[index].value) {
			continue
		}
		value := field.value.ValueBool()
		*field.target = &value
		changed = true
	}
	if !plan.KerberosEncryptionTypes.IsNull() && !plan.KerberosEncryptionTypes.Equal(current.KerberosEncryptionTypes) {
		request.Properties.KerberosEncryptionTypes, _ = encryptionTypes(plan.KerberosEncryptionTypes)
		changed = true
	}
	if !plan.CompoundIdentitySupported.IsNull() && !plan.CompoundIdentitySupported.Equal(current.CompoundIdentitySupported) {
		value := plan.CompoundIdentitySupported.ValueBool()
		request.Properties.CompoundIdentitySupported = &value
		changed = true
	}
	if !plan.AccountExpires.IsNull() && !plan.AccountExpires.Equal(current.AccountExpires) {
		value, _ := shared.ParseFileTime(strconv.FormatInt(plan.AccountExpires.ValueInt64(), 10))
		request.Properties.AccountExpires = &value
		changed = true
	}
	return request, changed
}

// retainAcceptedProperties records acknowledged selected intent before partial readback can hide its effect.
func retainAcceptedProperties(current, plan *ComputerResourceModel, request computers.UpdateComputerRequest) {
	before, after := textProperties(current, &request.Properties), textProperties(plan, &request.Properties)
	for index, field := range before {
		if *field.target != nil {
			*field.value = *after[index].value
		}
		for _, change := range request.Changes {
			if change.Kind == shared.AttributeClear && strings.EqualFold(change.Attribute, field.ldap) {
				*field.value = types.StringNull()
			}
		}
	}
	oldFlags, newFlags := boolProperties(current, &request.Control), boolProperties(plan, &request.Control)
	for index, field := range oldFlags {
		if *field.target != nil {
			*field.value = *newFlags[index].value
		}
	}
	if request.Properties.KerberosEncryptionTypes != nil {
		current.KerberosEncryptionTypes = plan.KerberosEncryptionTypes
	}
	if request.Properties.CompoundIdentitySupported != nil {
		current.CompoundIdentitySupported = plan.CompoundIdentitySupported
	}
	if request.Properties.AccountExpires != nil {
		current.AccountExpires = plan.AccountExpires
	}
	current.ClearProperties = plan.ClearProperties
}
