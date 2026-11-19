// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package users shares read-only projections and bounded discovery without owning passwords or account attributes.
package users

import (
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkusers "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// TextAttributes maps observable Terraform fields to individual LDAP text attributes.
func TextAttributes() map[string]string {
	return map[string]string{
		"sam_account_name": "sAMAccountName", "user_principal_name": "userPrincipalName", "given_name": "givenName", "surname": "sn",
		"display_name": "displayName", "description": "description", "email_address": "mail", "office_phone": "telephoneNumber", "mobile_phone": "mobile",
		"department": "department", "company": "company", "title": "title", "office": "physicalDeliveryOfficeName", "city": "l", "state": "st",
		"street_address": "streetAddress", "postal_code": "postalCode", "home_directory": "homeDirectory", "home_drive": "homeDrive", "profile_path": "profilePath",
		"script_path": "scriptPath", "manager_dn": "manager",
	}
}

// AttributeTypes excludes password input and provider-local version metadata from directory observations.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{}
	for _, name := range []string{"id", "name", "parent_dn", "object_class", "distinguished_name", "sid"} {
		result[name] = types.StringType
	}
	for name := range TextAttributes() {
		result[name] = types.StringType
	}
	for _, name := range []string{"enabled", "password_not_required", "password_never_expires", "smartcard_logon_required"} {
		result[name] = types.BoolType
	}
	result["account_expires"] = types.Int64Type
	return result
}

// Project emits usable GUID/location and nullable observations, never secret values or fabricated account flags.
func Project(user sdkusers.User) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if user.GUID == nil || *user.GUID == (shared.GUID{}) || user.Name == nil {
		diagnostics.AddError("Incomplete user identity", "A usable GUID and name are required. No import identity is emitted.")
		return nil, diagnostics
	}
	parent, valid := identity.ParentDN(user.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete user location", "A valid distinguished name and parent are required.")
		return nil, diagnostics
	}
	values := map[string]attr.Value{
		"id": types.StringValue(
			user.GUID.String(),
		),
		"name":               types.StringValue(*user.Name),
		"parent_dn":          types.StringValue(parent),
		"distinguished_name": types.StringValue(user.DistinguishedName),
		"sid":                types.StringNull(),
		"object_class":       types.StringNull(),
		"account_expires":    types.Int64Null(),
	}
	if user.SID != nil {
		values["sid"] = types.StringValue(user.SID.String())
	}
	classes, state := user.Raw.Attribute("objectClass")
	if state == shared.AttributePresent {
		for _, class := range classes {
			if strings.EqualFold(string(class), "inetOrgPerson") {
				values["object_class"] = types.StringValue("inetOrgPerson")
				break
			}
			if strings.EqualFold(string(class), "user") {
				values["object_class"] = types.StringValue("user")
			}
		}
	}
	for field, ldap := range TextAttributes() {
		observed := lifecycle.ObserveString(user.Raw, ldap)
		diagnostics.Append(observed.Diagnostics...)
		values[field] = observed.Value
	}
	for field, value := range map[string]*bool{
		"enabled": user.Enabled, "password_not_required": user.PasswordNotRequired,
		"password_never_expires": user.PasswordNeverExpires, "smartcard_logon_required": user.SmartcardLogonRequired,
	} {
		values[field] = types.BoolPointerValue(value)
	}
	if user.AccountExpires != nil {
		if *user.AccountExpires > math.MaxInt64 {
			diagnostics.AddError("Unsupported user expiration", "The observed accountExpires exceeds the supported nonnegative FILETIME range.")
		} else {
			values["account_expires"] = types.Int64Value(int64(*user.AccountExpires))
		}
	}
	return values, diagnostics
}
