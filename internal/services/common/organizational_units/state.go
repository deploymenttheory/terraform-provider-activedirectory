// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package organizationalUnits provides shared read-only OU projections and explicit search construction.
package organizationalUnits

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// AttributeTypes defines the same observable OU fields for singular, plural and import discovery results.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{}
	for _, name := range []string{
		"id", "name", "parent_dn", "distinguished_name", "description", "display_name", "city", "country", "state", "street_address", "postal_code", "managed_by",
	} {
		result[name] = types.StringType
	}
	result["protected_from_accidental_deletion"] = types.BoolType
	return result
}

// Project returns only usable identity/location and preserves unavailable optional attributes as null.
func Project(unit organizational_units.OrganizationalUnit) (map[string]attr.Value, diag.Diagnostics) {
	values := map[string]attr.Value{}
	var diagnostics diag.Diagnostics
	if unit.GUID == nil || *unit.GUID == (shared.GUID{}) || unit.Name == nil {
		diagnostics.AddError("Incomplete OU identity", "The directory did not return a usable object GUID and name. No import identity is emitted.")
		return nil, diagnostics
	}
	parent, ok := ParentDN(unit.DistinguishedName)
	if !ok {
		diagnostics.AddError("Incomplete OU location", "The directory did not return a usable distinguished name and parent.")
		return nil, diagnostics
	}
	values["id"] = types.StringValue(unit.GUID.String())
	values["name"] = types.StringValue(*unit.Name)
	values["parent_dn"] = types.StringValue(parent)
	values["distinguished_name"] = types.StringValue(unit.DistinguishedName)
	for field, ldap := range map[string]string{
		"description": "description", "display_name": "displayName", "city": "l", "country": "c",
		"state": "st", "street_address": "street", "postal_code": "postalCode", "managed_by": "managedBy",
	} {
		observed := lifecycle.ObserveString(unit.Raw, ldap)
		diagnostics.Append(observed.Diagnostics...)
		values[field] = observed.Value
	}
	values["protected_from_accidental_deletion"] = types.BoolPointerValue(unit.ProtectedFromAccidentalDeletion)
	return values, diagnostics
}

// ParentDN locates an unescaped RFC4514 separator after SDK validation, preserving directory wire spelling.
func ParentDN(dn string) (string, bool) {
	if _, err := shared.IdentityByDN(dn); err != nil {
		return "", false
	}
	for index := 0; index < len(dn); index++ {
		if dn[index] == '\\' {
			index++
			continue
		}
		if dn[index] == ',' {
			return dn[index+1:], true
		}
	}
	return "", false
}
