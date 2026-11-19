// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

type property struct {
	ldap   string
	value  *types.String
	target **string
}

// propertyBindings maps Terraform fields to explicit SDK property construction and raw observations.
func propertyBindings(model *OrganizationalUnitResourceModel, properties *organizational_units.OUProperties) []property {
	return []property{
		{"description", &model.Description, &properties.Description},
		{"displayName", &model.DisplayName, &properties.DisplayName},
		{"l", &model.City, &properties.City},
		{"c", &model.Country, &properties.Country},
		{"st", &model.State, &properties.State},
		{"street", &model.StreetAddress, &properties.StreetAddress},
		{"postalCode", &model.PostalCode, &properties.PostalCode},
		{"managedBy", &model.ManagedBy, &properties.ManagedBy},
	}
}

// validatePlan refuses unknown/malformed inputs before any phase of a multi-operation update.
func validatePlan(model *OrganizationalUnitResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if model.Name.IsNull() || model.Name.IsUnknown() || model.Name.ValueString() == "" {
		diagnostics.AddError("Invalid OU name", "A known, nonempty literal OU name is required.")
	} else if !utf8.ValidString(model.Name.ValueString()) || strings.ContainsRune(model.Name.ValueString(), 0) {
		diagnostics.AddError("Invalid OU name", "The OU name must be valid directory text without NUL bytes.")
	}
	if model.ParentDN.IsNull() || model.ParentDN.IsUnknown() {
		diagnostics.AddError("Unknown OU parent", "The parent distinguished name must be known before any directory mutation.")
	} else if _, err := shared.IdentityByDN(model.ParentDN.ValueString()); err != nil {
		diagnostics.AddError("Invalid OU parent", "The parent must be an explicit valid distinguished name.")
	}
	if model.ProtectedFromAccidentalDeletion.IsNull() || model.ProtectedFromAccidentalDeletion.IsUnknown() {
		diagnostics.AddError("Unknown OU protection", "Deletion protection must be known before any directory mutation.")
	}
	properties := organizational_units.OUProperties{}
	for _, field := range propertyBindings(model, &properties) {
		_, fieldDiagnostics := lifecycle.PlannedString(*field.value)
		diagnostics.Append(fieldDiagnostics...)
		if !field.value.IsNull() && !field.value.IsUnknown() &&
			(!utf8.ValidString(field.value.ValueString()) || strings.ContainsRune(field.value.ValueString(), 0)) {
			diagnostics.AddError("Invalid OU property", "Managed properties must be valid directory text without NUL bytes.")
		}
	}
	if !model.ManagedBy.IsNull() && !model.ManagedBy.IsUnknown() {
		if _, err := shared.IdentityByDN(model.ManagedBy.ValueString()); err != nil {
			diagnostics.AddError("Invalid OU manager", "managed_by must be an explicit valid distinguished name.")
		}
	}
	return diagnostics
}

// constructCreate builds an SDK Add request; ordinary writes leave ExpectedVersion unset.
func constructCreate(model *OrganizationalUnitResourceModel) (organizational_units.CreateOrganizationalUnitRequest, diag.Diagnostics) {
	diagnostics := validatePlan(model)
	if diagnostics.HasError() {
		return organizational_units.CreateOrganizationalUnitRequest{}, diagnostics
	}
	parent, _ := shared.IdentityByDN(model.ParentDN.ValueString())
	request := organizational_units.CreateOrganizationalUnitRequest{Parent: parent, Name: model.Name.ValueString()}
	protect := model.ProtectedFromAccidentalDeletion.ValueBool()
	request.ProtectedFromAccidentalDeletion = &protect
	for _, field := range propertyBindings(model, &request.Properties) {
		value, _ := lifecycle.PlannedString(*field.value)
		*field.target = value
	}
	return request, diagnostics
}

// constructUpdate owns only changed configured properties; removing a managed value is an explicit clear.
func constructUpdate(current, plan *OrganizationalUnitResourceModel, target shared.Identity) organizational_units.UpdateOrganizationalUnitRequest {
	request := organizational_units.UpdateOrganizationalUnitRequest{Identity: target}
	ignored := organizational_units.OUProperties{}
	before, after := propertyBindings(current, &ignored), propertyBindings(plan, &request.Properties)
	previousClears, _ := clearProperties(current.ClearProperties)
	requestedClears, _ := clearProperties(plan.ClearProperties)
	for index, field := range after {
		newClear := requestedClears[strings.ToLower(field.ldap)] && !previousClears[strings.ToLower(field.ldap)]
		if field.value.Equal(*before[index].value) && !newClear {
			continue
		}
		if field.value.IsNull() {
			request.Changes = append(request.Changes, shared.AttributeChange{Attribute: field.ldap, Kind: shared.AttributeClear})
		} else {
			value := field.value.ValueString()
			*field.target = &value
		}
	}
	if !current.ProtectedFromAccidentalDeletion.Equal(plan.ProtectedFromAccidentalDeletion) {
		protect := plan.ProtectedFromAccidentalDeletion.ValueBool()
		request.ProtectedFromAccidentalDeletion = &protect
	}
	return request
}

// hasPropertyChanges avoids submitting an empty Modify during rename/move-only updates.
func hasPropertyChanges(current, plan *OrganizationalUnitResourceModel) bool {
	ignored := organizational_units.OUProperties{}
	before, after := propertyBindings(current, &ignored), propertyBindings(plan, &ignored)
	previousClears, _ := clearProperties(current.ClearProperties)
	requestedClears, _ := clearProperties(plan.ClearProperties)
	for index, field := range after {
		if !field.value.Equal(*before[index].value) || (requestedClears[strings.ToLower(field.ldap)] && !previousClears[strings.ToLower(field.ldap)]) {
			return true
		}
	}
	return !current.ProtectedFromAccidentalDeletion.Equal(plan.ProtectedFromAccidentalDeletion)
}
