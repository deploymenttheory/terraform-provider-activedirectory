// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// groupCategory accepts an explicit semantic category without constructing a raw groupType replacement mask.
func groupCategory(value types.String) (groups.GroupCategory, diag.Diagnostics) {
	switch value.ValueString() {
	case "security":
		return groups.GroupSecurity, nil
	case "distribution":
		return groups.GroupDistribution, nil
	default:
		return groups.GroupCategoryUnknown, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid group category", "category must be a known security or distribution value.")}
	}
}

// groupScope leaves server-authoritative conversion constraints to the SDK and selected directory.
func groupScope(value types.String) (groups.GroupScope, diag.Diagnostics) {
	switch value.ValueString() {
	case "global":
		return groups.GroupGlobal, nil
	case "domain_local":
		return groups.GroupDomainLocal, nil
	case "universal":
		return groups.GroupUniversal, nil
	default:
		return groups.GroupScopeUnknown, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid group scope", "scope must be a known global, domain_local or universal value.")}
	}
}

// constructCreate validates explicit identity and typed properties before permitting any group creation.
func constructCreate(model *GroupResourceModel) (groups.CreateGroupRequest, diag.Diagnostics) {
	var request groups.CreateGroupRequest
	var diagnostics diag.Diagnostics
	for _, field := range []struct {
		name  string
		value types.String
	}{
		{"name", model.Name}, {"sam_account_name", model.SAMAccountName}, {"parent_dn", model.ParentDN},
	} {
		if field.value.IsUnknown() || field.value.IsNull() || strings.TrimSpace(field.value.ValueString()) == "" || !validText(field.value.ValueString()) {
			diagnostics.AddError("Invalid group identity", field.name+" must be known, nonempty directory text.")
		}
	}
	request.Name = model.Name.ValueString()
	var err error
	request.Parent, err = shared.IdentityByDN(model.ParentDN.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid group parent", "parent_dn must be an explicit distinguished name. No default container is selected.")
	}
	var categoryDiagnostics, scopeDiagnostics diag.Diagnostics
	request.Category, categoryDiagnostics = groupCategory(model.Category)
	request.Scope, scopeDiagnostics = groupScope(model.Scope)
	diagnostics.Append(categoryDiagnostics...)
	diagnostics.Append(scopeDiagnostics...)
	for _, field := range []struct {
		value  types.String
		target **string
	}{
		{model.SAMAccountName, &request.SAMAccountName},
		{model.Description, &request.Description},
		{model.DisplayName, &request.DisplayName},
		{model.HomePage, &request.HomePage},
		{model.ManagedByDN, &request.ManagedByDN},
	} {
		if field.value.IsUnknown() {
			diagnostics.AddError("Unknown group property", "Configured group properties must be known before submitting a change.")
		} else if !field.value.IsNull() {
			value := field.value.ValueString()
			if !validText(value) {
				diagnostics.AddError("Invalid group property text", "Group properties must be valid UTF-8 directory text without NUL bytes.")
			} else {
				*field.target = &value
			}
		}
	}
	if request.ManagedByDN != nil {
		if _, err := shared.IdentityByDN(*request.ManagedByDN); err != nil {
			diagnostics.AddError("Invalid group manager", "managed_by_dn must be a distinguished name; no implicit principal lookup is performed.")
		}
	}
	if model.ClearProperties.IsUnknown() || (!model.ClearProperties.IsNull() && len(model.ClearProperties.Elements()) > 0) {
		diagnostics.AddError("Group clear requires an existing object", "clear_properties applies to an existing or imported group.")
	}
	if diagnostics.HasError() {
		return groups.CreateGroupRequest{}, diagnostics
	}
	return request, diagnostics
}

// validText rejects malformed LDAP text before request construction.
func validText(value string) bool { return utf8.ValidString(value) && !strings.ContainsRune(value, 0) }
