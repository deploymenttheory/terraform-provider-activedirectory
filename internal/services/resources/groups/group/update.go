// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type textProperty struct {
	ldap   string
	value  *types.String
	target **string
}

// propertyBindings keeps individual typed changes separate from groupType and membership ownership.
func propertyBindings(model *GroupResourceModel, request *groups.UpdateGroupRequest) []textProperty {
	return []textProperty{
		{"sAMAccountName", &model.SAMAccountName, &request.SAMAccountName},
		{"description", &model.Description, &request.Description},
		{"displayName", &model.DisplayName, &request.DisplayName},
		{"wWWHomePage", &model.HomePage, &request.HomePage},
		{"managedBy", &model.ManagedByDN, &request.ManagedByDN},
	}
}

// clearProperties refuses account identity, membership and flag attributes before constructing any mutation.
func clearProperties(value types.Set) (map[string]bool, diag.Diagnostics) {
	result := map[string]bool{}
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown group clear intent", "Resolve clear_properties before any mutation.")
		return result, diagnostics
	}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsNull() || text.IsUnknown() {
			diagnostics.AddError("Invalid group clear intent", "Every clear_properties element must be a known attribute name.")
			continue
		}
		name := strings.ToLower(text.ValueString())
		switch name {
		case "description", "displayname", "wwwhomepage", "managedby":
			result[name] = true
		default:
			diagnostics.AddError("Unsupported group property clear", "Only description, displayName, wWWHomePage and managedBy may be explicitly cleared.")
		}
	}
	return result, diagnostics
}

// validatePlan checks all property, naming and clearing intent before an earlier rename or move could be accepted.
func validatePlan(model *GroupResourceModel) diag.Diagnostics {
	creation := *model
	creation.ClearProperties = types.SetNull(types.StringType)
	_, diagnostics := constructCreate(&creation)
	clears, clearDiagnostics := clearProperties(model.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	ignored := groups.UpdateGroupRequest{}
	for _, field := range propertyBindings(model, &ignored) {
		if clears[strings.ToLower(field.ldap)] && !field.value.IsNull() {
			diagnostics.AddError("Conflicting group property intent", "A property cannot have a configured value and an explicit clear at the same time.")
		}
	}
	return diagnostics
}

// constructUpdate emits only changed typed fields and new clear intent, never a replacement groupType or member collection.
func constructUpdate(current, plan *GroupResourceModel, target shared.Identity) (groups.UpdateGroupRequest, bool, diag.Diagnostics) {
	diagnostics := validatePlan(plan)
	if target.Form() != shared.IdentityFormGUID {
		diagnostics.AddError("Invalid group update identity", "Updates require the retained group GUID rather than a mutable DN.")
	}
	if diagnostics.HasError() {
		return groups.UpdateGroupRequest{}, false, diagnostics
	}
	request, ignored := groups.UpdateGroupRequest{Identity: target}, groups.UpdateGroupRequest{}
	before, after := propertyBindings(current, &ignored), propertyBindings(plan, &request)
	previousClears, _ := clearProperties(current.ClearProperties)
	requestedClears, _ := clearProperties(plan.ClearProperties)
	changed := false
	for index, field := range after {
		clearRequested := requestedClears[strings.ToLower(field.ldap)]
		newClear := clearRequested && !previousClears[strings.ToLower(field.ldap)]
		if field.value.IsNull() && !clearRequested || field.value.Equal(*before[index].value) && !newClear {
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
	if !current.Category.Equal(plan.Category) {
		value, _ := groupCategory(plan.Category)
		request.Category, changed = &value, true
	}
	if !current.Scope.Equal(plan.Scope) {
		value, _ := groupScope(plan.Scope)
		request.Scope, changed = &value, true
	}
	return request, changed, diagnostics
}
