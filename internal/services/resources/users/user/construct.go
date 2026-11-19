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

type textProperty struct {
	terraform string
	ldap      string
	value     *types.String
	target    **string
}

// textProperties is the explicit mapping between Terraform properties and the typed SDK request.
func textProperties(model *UserResourceModel, properties *users.UserProperties) []textProperty {
	return []textProperty{
		{"sam_account_name", "sAMAccountName", &model.SAMAccountName, &properties.SAMAccountName},
		{"user_principal_name", "userPrincipalName", &model.UserPrincipalName, &properties.UserPrincipalName},
		{"given_name", "givenName", &model.GivenName, &properties.GivenName},
		{"surname", "sn", &model.Surname, &properties.Surname},
		{"display_name", "displayName", &model.DisplayName, &properties.DisplayName},
		{"description", "description", &model.Description, &properties.Description},
		{"email_address", "mail", &model.EmailAddress, &properties.EmailAddress},
		{"office_phone", "telephoneNumber", &model.OfficePhone, &properties.OfficePhone},
		{"mobile_phone", "mobile", &model.MobilePhone, &properties.MobilePhone},
		{"department", "department", &model.Department, &properties.Department},
		{"company", "company", &model.Company, &properties.Company},
		{"title", "title", &model.Title, &properties.Title},
		{"office", "physicalDeliveryOfficeName", &model.Office, &properties.Office},
		{"city", "l", &model.City, &properties.City},
		{"state", "st", &model.State, &properties.State},
		{"street_address", "streetAddress", &model.StreetAddress, &properties.StreetAddress},
		{"postal_code", "postalCode", &model.PostalCode, &properties.PostalCode},
		{"home_directory", "homeDirectory", &model.HomeDirectory, &properties.HomeDirectory},
		{"home_drive", "homeDrive", &model.HomeDrive, &properties.HomeDrive},
		{"profile_path", "profilePath", &model.ProfilePath, &properties.ProfilePath},
		{"script_path", "scriptPath", &model.ScriptPath, &properties.ScriptPath},
		{"manager_dn", "manager", &model.ManagerDN, &properties.ManagerDN},
	}
}

// constructProperties retains null omission and literal empty strings, refusing unresolved writes.
func constructProperties(model *UserResourceModel) (users.UserProperties, diag.Diagnostics) {
	var properties users.UserProperties
	var diagnostics diag.Diagnostics
	for _, field := range textProperties(model, &properties) {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown user property", field.terraform+" must be known before submitting a user change.")
		} else if !field.value.IsNull() {
			value := field.value.ValueString()
			if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
				diagnostics.AddError("Invalid user property text", "Typed user properties must be valid directory text without NUL bytes.")
				continue
			}
			*field.target = &value
		}
	}
	for _, field := range []struct {
		value  types.Bool
		target **bool
	}{
		{model.Enabled, &properties.Enabled},
		{model.PasswordNotRequired, &properties.PasswordNotRequired},
		{model.PasswordNeverExpires, &properties.PasswordNeverExpires},
		{model.SmartcardLogonRequired, &properties.SmartcardLogonRequired},
	} {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown user account control", "Account-control properties must be known before submitting a change.")
		} else if !field.value.IsNull() {
			value := field.value.ValueBool()
			*field.target = &value
		}
	}
	stamp := model.AccountExpires.ValueInt64()
	if model.AccountExpires.IsUnknown() || (!model.AccountExpires.IsNull() && stamp < 0) {
		diagnostics.AddError("Invalid user expiration", "account_expires must be a known nonnegative FILETIME, including the explicit never-expire sentinel zero.")
	} else if !model.AccountExpires.IsNull() {
		value, err := shared.ParseFileTime(strconv.FormatInt(stamp, 10))
		if err != nil {
			diagnostics.AddError("Invalid user expiration encoding", "The SDK rejected the explicit FILETIME value.")
		} else {
			properties.AccountExpires = &value
		}
	}
	if properties.ManagerDN != nil {
		if _, err := shared.IdentityByDN(*properties.ManagerDN); err != nil {
			diagnostics.AddError("Invalid user manager", "manager_dn must be an explicit distinguished name.")
		}
	}
	return properties, diagnostics
}

// constructCreate validates the whole request before allocating a caller-owned password buffer.
// The SDK creates disabled, optionally sets the initial password and then enables only when explicitly requested.
func constructCreate(model *UserResourceModel, password types.String) (users.CreateUserRequest, diag.Diagnostics) {
	properties, diagnostics := constructProperties(model)
	request := users.CreateUserRequest{Name: model.Name.ValueString(), Class: model.ObjectClass.ValueString(), Properties: properties}
	if model.Name.IsNull() || model.Name.IsUnknown() || strings.TrimSpace(request.Name) == "" || model.SAMAccountName.IsNull() ||
		model.SAMAccountName.IsUnknown() || strings.TrimSpace(model.SAMAccountName.ValueString()) == "" {
		diagnostics.AddError("Unknown user name", "Provide known, nonempty name and sam_account_name before creating a user.")
	}
	if model.ParentDN.IsNull() || model.ParentDN.IsUnknown() {
		diagnostics.AddError("Unknown user parent", "An explicit known parent_dn is required. No default directory container is selected.")
	} else {
		var err error
		request.Parent, err = shared.IdentityByDN(model.ParentDN.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid user parent", "parent_dn must be a distinguished name.")
		}
	}
	if model.ObjectClass.IsNull() {
		request.Class = "user"
	} else if model.ObjectClass.IsUnknown() || (request.Class != "user" && request.Class != "inetOrgPerson") {
		diagnostics.AddError("Invalid user class", "object_class must be a known user or inetOrgPerson schema class.")
	}
	diagnostics.Append(validateInitialPassword(model.PasswordWOVersion, password)...)
	if properties.Enabled != nil && *properties.Enabled && password.IsNull() && (properties.PasswordNotRequired == nil || !*properties.PasswordNotRequired) {
		diagnostics.AddError("Initial password required", "Enabling a newly created user requires password setup or explicitly acknowledged password_not_required=true.")
	}
	if !model.ClearProperties.IsNull() && !model.ClearProperties.IsUnknown() && len(model.ClearProperties.Elements()) > 0 {
		diagnostics.AddError("Cannot clear properties during creation", "clear_properties applies only to an existing user.")
	}
	if model.ClearProperties.IsUnknown() {
		diagnostics.AddError("Unknown clear intent", "Resolve clear_properties before creating a user.")
	}
	if diagnostics.HasError() {
		return users.CreateUserRequest{}, diagnostics
	}
	if !password.IsNull() {
		request.InitialPassword = []byte(password.ValueString())
	}
	return request, diagnostics
}

// validateInitialPassword requires an explicit, known version whenever a new secret is supplied.
func validateInitialPassword(version types.Int64, password types.String) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if version.IsUnknown() || version.ValueInt64() < 0 || password.IsUnknown() {
		diagnostics.AddError("Unknown password intent", "Password input and its nonnegative version must be known before user creation.")
	}
	if !password.IsNull() &&
		(password.ValueString() == "" || !utf8.ValidString(password.ValueString()) || strings.ContainsRune(password.ValueString(), 0) || version.ValueInt64() < 1) {
		diagnostics.AddError("Invalid password intent", "A nonempty password_wo requires an explicit positive password_wo_version.")
	}
	if password.IsNull() && version.ValueInt64() > 0 {
		diagnostics.AddError("Missing initial password", "A positive password_wo_version requires password_wo for creation. Secret values are never recovered from state.")
	}
	return diagnostics
}
