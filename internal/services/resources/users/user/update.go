// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// validateUpdate checks every requested phase before an earlier phase can have accepted effects.
func validateUpdate(current, plan *UserResourceModel, password types.String) diag.Diagnostics {
	_, diagnostics := constructProperties(plan)
	if plan.Name.IsNull() || plan.Name.IsUnknown() || strings.TrimSpace(plan.Name.ValueString()) == "" || strings.ContainsRune(plan.Name.ValueString(), 0) {
		diagnostics.AddError("Invalid user name", "A known nonempty literal name without NUL is required.")
	}
	if plan.SAMAccountName.IsNull() || plan.SAMAccountName.IsUnknown() || plan.SAMAccountName.ValueString() == "" {
		diagnostics.AddError("Invalid user account name", "A known nonempty sam_account_name is required.")
	}
	if plan.ParentDN.IsNull() || plan.ParentDN.IsUnknown() {
		diagnostics.AddError("Unknown user parent", "Resolve the explicit parent_dn before any mutation.")
	} else if _, err := shared.IdentityByDN(plan.ParentDN.ValueString()); err != nil {
		diagnostics.AddError("Invalid user parent", "parent_dn must be a distinguished name.")
	}
	if !current.ObjectClass.Equal(plan.ObjectClass) {
		diagnostics.AddError("User class change requires replacement", "The existing object class cannot be changed by an update.")
	}
	if plan.PasswordWOVersion.IsNull() || plan.PasswordWOVersion.IsUnknown() || plan.PasswordWOVersion.ValueInt64() < current.PasswordWOVersion.ValueInt64() {
		diagnostics.AddError("Invalid user password version", "password_wo_version must retain or increase the acknowledged version.")
	} else if plan.PasswordWOVersion.ValueInt64() > current.PasswordWOVersion.ValueInt64() {
		if password.IsUnknown() || password.IsNull() || password.ValueString() == "" || !utf8.ValidString(password.ValueString()) ||
			strings.ContainsRune(password.ValueString(), 0) {
			diagnostics.AddError("New password input required", "An increased password_wo_version requires a known nonempty password_wo before any phase begins.")
		}
	}
	return diagnostics
}

// constructUpdate omits unmanaged/unchanged fields and emits a clear only for explicit clear intent.
func constructUpdate(current, plan *UserResourceModel, target shared.Identity) (users.UpdateUserRequest, bool) {
	request := users.UpdateUserRequest{Identity: target}
	ignored := users.UserProperties{}
	before, after := textProperties(current, &ignored), textProperties(plan, &request.Properties)
	previousClears, _ := clearProperties(current.ClearProperties)
	requestedClears, _ := clearProperties(plan.ClearProperties)
	changed := false
	for index, field := range after {
		newClear := requestedClears[strings.ToLower(field.ldap)] && !previousClears[strings.ToLower(field.ldap)]
		if field.value.IsNull() && !requestedClears[strings.ToLower(field.ldap)] || (field.value.Equal(*before[index].value) && !newClear) {
			continue
		}
		changed = true
		if field.value.IsNull() {
			request.Changes = append(request.Changes, shared.AttributeChange{Kind: shared.AttributeClear, Attribute: field.ldap})
		} else {
			value := field.value.ValueString()
			*field.target = &value
		}
	}
	oldFlags, newFlags := boolProperties(current, &ignored), boolProperties(plan, &request.Properties)
	for index, field := range newFlags {
		if field.value.IsNull() || field.value.Equal(*oldFlags[index].value) {
			continue
		}
		value := field.value.ValueBool()
		*field.target = &value
		changed = true
	}
	if !plan.AccountExpires.IsNull() && !current.AccountExpires.Equal(plan.AccountExpires) {
		value, _ := shared.ParseFileTime(strconv.FormatInt(plan.AccountExpires.ValueInt64(), 10))
		request.Properties.AccountExpires = &value
		changed = true
	}
	return request, changed
}

// retainAcceptedProperties preserves acknowledged clear intent before later readback can hide its effect.
func retainAcceptedProperties(current, plan *UserResourceModel, request users.UpdateUserRequest) {
	ignored := users.UserProperties{}
	before, after := textProperties(current, &ignored), textProperties(plan, &ignored)
	for index, field := range after {
		if !field.value.IsNull() {
			*before[index].value = *field.value
		}
	}
	for _, change := range request.Changes {
		for _, field := range before {
			if change.Kind == shared.AttributeClear && strings.EqualFold(change.Attribute, field.ldap) {
				*field.value = types.StringNull()
			}
		}
	}
	oldFlags, newFlags := boolProperties(current, &ignored), boolProperties(plan, &ignored)
	for index, field := range newFlags {
		if !field.value.IsNull() {
			*oldFlags[index].value = *field.value
		}
	}
	if !plan.AccountExpires.IsNull() {
		current.AccountExpires = plan.AccountExpires
	}
	current.ClearProperties = plan.ClearProperties
}
