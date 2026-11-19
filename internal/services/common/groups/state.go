// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package groups shares bounded discovery and read-only projections without owning group membership.
package groups

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkgroups "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// TextAttributes maps each observable Terraform field to its individual LDAP text attribute.
func TextAttributes() map[string]string {
	return map[string]string{
		"sam_account_name": "sAMAccountName", "description": "description", "display_name": "displayName", "home_page": "wWWHomePage", "managed_by_dn": "managedBy",
	}
}

// AttributeTypes describes the shared resource/data/import projection without writable raw groupType or member fields.
func AttributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{}
	for _, name := range []string{"id", "name", "parent_dn", "distinguished_name", "sid", "category", "scope"} {
		result[name] = types.StringType
	}
	for name := range TextAttributes() {
		result[name] = types.StringType
	}
	return result
}

// Project preserves nullable observations and refuses unusable identities instead of emitting an invalid import target.
func Project(group sdkgroups.Group) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if group.GUID == nil || *group.GUID == (shared.GUID{}) || group.Name == nil {
		diagnostics.AddError("Incomplete group identity", "A usable GUID and name are required. No import identity is emitted.")
		return nil, diagnostics
	}
	parent, valid := identity.ParentDN(group.DistinguishedName)
	if !valid {
		diagnostics.AddError("Incomplete group location", "A valid directory-observed distinguished name and parent are required.")
		return nil, diagnostics
	}
	values := map[string]attr.Value{
		"id": types.StringValue(group.GUID.String()), "name": types.StringValue(*group.Name), "parent_dn": types.StringValue(parent),
		"distinguished_name": types.StringValue(group.DistinguishedName), "sid": types.StringNull(), "category": types.StringNull(), "scope": types.StringNull(),
	}
	if group.SID != nil {
		values["sid"] = types.StringValue(group.SID.String())
	}
	for field, ldap := range TextAttributes() {
		value := lifecycle.ObserveString(group.Raw, ldap)
		diagnostics.Append(value.Diagnostics...)
		values[field] = value.Value
	}
	if group.Category != nil {
		switch *group.Category {
		case sdkgroups.GroupSecurity:
			values["category"] = types.StringValue("security")
		case sdkgroups.GroupDistribution:
			values["category"] = types.StringValue("distribution")
		}
	}
	if group.Scope != nil {
		switch *group.Scope {
		case sdkgroups.GroupGlobal:
			values["scope"] = types.StringValue("global")
		case sdkgroups.GroupDomainLocal:
			values["scope"] = types.StringValue("domain_local")
		case sdkgroups.GroupUniversal:
			values["scope"] = types.StringValue("universal")
		}
	}
	return values, diagnostics
}
