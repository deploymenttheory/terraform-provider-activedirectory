// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// ownedPlan excludes observed but unconfigured properties from writes after earlier phases refresh concurrent changes.
func ownedPlan(plan, config *ServiceAccountResourceModel) ServiceAccountResourceModel {
	owned := *plan
	properties, control := service_accounts.ServiceAccountProperties{}, accounts.AccountControlPatch{}
	planned, configured := textProperties(&owned, &properties), textProperties(config, &properties)
	for index, field := range planned {
		if configured[index].value.IsNull() {
			*field.value = types.StringNull()
		}
	}
	plannedFlags, configuredFlags := boolProperties(&owned, &control), boolProperties(config, &control)
	for index, field := range plannedFlags {
		if configuredFlags[index].value.IsNull() {
			*field.value = types.BoolNull()
		}
	}
	for planned, configured := range map[*types.Set]*types.Set{
		&owned.KerberosEncryptionTypes: &config.KerberosEncryptionTypes,
		&owned.ServicePrincipalNames:   &config.ServicePrincipalNames, &owned.PasswordReaderSIDs: &config.PasswordReaderSIDs,
	} {
		if configured.IsNull() {
			*planned = types.SetNull(types.StringType)
		}
	}
	if config.CompoundIdentitySupported.IsNull() {
		owned.CompoundIdentitySupported = types.BoolNull()
	}
	if config.AccountExpires.IsNull() {
		owned.AccountExpires = types.Int64Null()
	}
	return owned
}

// validateUpdate refuses replacement-only inputs and validates every later phase before any mutation.
func validateUpdate(current, plan *ServiceAccountResourceModel, _ types.String) diag.Diagnostics {
	_, _, diagnostics := constructProperties(plan)
	diagnostics.Append(validateNames(plan)...)
	if !plan.Kind.Equal(current.Kind) || !plan.OutboundAuthenticationOnly.Equal(current.OutboundAuthenticationOnly) ||
		!plan.ManagedPasswordIntervalDays.Equal(current.ManagedPasswordIntervalDays) || !plan.PasswordWOVersion.Equal(current.PasswordWOVersion) {
		diagnostics.AddError("Service account replacement required", "Kind, outbound creation form, password interval and creation password version cannot be updated in place.")
	}
	_, clearing := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearing...)
	return diagnostics
}

// constructUpdate emits selected changed properties, explicit clears and semantic account/encryption patches; it never replaces a raw mask.
func constructUpdate(current, plan *ServiceAccountResourceModel, target shared.Identity) (service_accounts.UpdateServiceAccountRequest, bool) {
	request := service_accounts.UpdateServiceAccountRequest{Identity: target}
	ignored := service_accounts.ServiceAccountProperties{}
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
	if !plan.ServicePrincipalNames.IsNull() && !plan.ServicePrincipalNames.Equal(current.ServicePrincipalNames) {
		request.Properties.ServicePrincipalNames, _ = stringValues(plan.ServicePrincipalNames)
		changed = true
	}
	if !plan.PasswordReaderSIDs.IsNull() && !plan.PasswordReaderSIDs.Equal(current.PasswordReaderSIDs) {
		request.PasswordReaders, _ = constructReaders(plan.PasswordReaderSIDs)
		changed = true
	}
	return request, changed
}

// retainAcceptedProperties records acknowledged selected intent before partial readback can hide its effect.
func retainAcceptedProperties(current, plan *ServiceAccountResourceModel, request service_accounts.UpdateServiceAccountRequest) {
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
	if request.Properties.ServicePrincipalNames != nil {
		current.ServicePrincipalNames = plan.ServicePrincipalNames
	}
	if request.PasswordReaders != nil {
		current.PasswordReaderSIDs = plan.PasswordReaderSIDs
	}
	current.ClearProperties = plan.ClearProperties
}
