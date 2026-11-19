// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
)

// modifyServiceAccountPlan exposes selected clears and permits concurrent changes to unowned observations.
func modifyServiceAccountPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config ServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.PasswordWOVersion.IsUnknown() && plan.PasswordWOVersion.ValueInt64() < 0 {
		resp.Diagnostics.AddError("Invalid creation password version", "password_wo_version must be nonnegative. Changes require replacement.")
	}
	if !req.State.Raw.IsNull() && !req.Plan.Raw.Equal(req.State.Raw) {
		markUnownedUnknown(&plan, &config)
		var previous ServiceAccountResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
		if config.ManagedPasswordIntervalDays.IsNull() && (!plan.Kind.Equal(previous.Kind) || !plan.OutboundAuthenticationOnly.Equal(previous.OutboundAuthenticationOnly)) {
			plan.ManagedPasswordIntervalDays = types.Int64Unknown()
		}
	}
	if plan.ClearProperties.IsUnknown() {
		return
	}
	clears, diagnostics := clearProperties(plan.ClearProperties)
	resp.Diagnostics.Append(diagnostics...)
	if req.State.Raw.IsNull() && len(clears) > 0 {
		resp.Diagnostics.AddError("Service account clear requires an existing object", "Explicit property clearing applies only to an existing or imported service_account.")
	}
	ignored := service_accounts.ServiceAccountProperties{}
	planned, configured := textProperties(&plan, &ignored), textProperties(&config, &ignored)
	for index, field := range planned {
		if !clears[strings.ToLower(field.ldap)] {
			continue
		}
		if !configured[index].value.IsNull() {
			resp.Diagnostics.AddError("Conflicting service account property intent", "A property cannot have a configured value and be explicitly cleared at the same time.")
		} else {
			*field.value = types.StringNull()
		}
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// markUnownedUnknown allows refreshed concurrent observations during an update without planning them as managed values.
func markUnownedUnknown(plan, config *ServiceAccountResourceModel) {
	properties, control := service_accounts.ServiceAccountProperties{}, accounts.AccountControlPatch{}
	planned, configured := textProperties(plan, &properties), textProperties(config, &properties)
	for index, field := range planned {
		if configured[index].value.IsNull() {
			*field.value = types.StringUnknown()
		}
	}
	plannedFlags, configuredFlags := boolProperties(plan, &control), boolProperties(config, &control)
	for index, field := range plannedFlags {
		if configuredFlags[index].value.IsNull() {
			*field.value = types.BoolUnknown()
		}
	}
	for planned, configured := range map[*types.Set]*types.Set{
		&plan.KerberosEncryptionTypes: &config.KerberosEncryptionTypes,
		&plan.ServicePrincipalNames:   &config.ServicePrincipalNames, &plan.PasswordReaderSIDs: &config.PasswordReaderSIDs,
	} {
		if configured.IsNull() {
			*planned = types.SetUnknown(types.StringType)
		}
	}
	if config.CompoundIdentitySupported.IsNull() {
		plan.CompoundIdentitySupported = types.BoolUnknown()
	}
	if config.AccountExpires.IsNull() {
		plan.AccountExpires = types.Int64Unknown()
	}
}

// preparePlan resolves provider-generated unknowns only; unknown configured inputs never authorize mutation.
func preparePlan(plan, config *ServiceAccountResourceModel, previous *ServiceAccountResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	properties, control := service_accounts.ServiceAccountProperties{}, accounts.AccountControlPatch{}
	planned, configured := textProperties(plan, &properties), textProperties(config, &properties)
	for index, field := range planned {
		if configured[index].value.IsUnknown() {
			diagnostics.AddError("Unknown service account property", "Resolve configured properties before a mutation.")
		} else if configured[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	plannedFlags, configuredFlags := boolProperties(plan, &control), boolProperties(config, &control)
	for index, field := range plannedFlags {
		if configuredFlags[index].value.IsUnknown() {
			diagnostics.AddError("Unknown service account account setting", "Resolve configured account settings before a mutation.")
		} else if configuredFlags[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	if config.CompoundIdentitySupported.IsUnknown() {
		diagnostics.AddError("Unknown compound identity setting", "Resolve compound_identity_supported before a mutation.")
	} else if config.CompoundIdentitySupported.IsNull() && plan.CompoundIdentitySupported.IsUnknown() {
		plan.CompoundIdentitySupported = types.BoolNull()
	}
	if config.KerberosEncryptionTypes.IsUnknown() {
		diagnostics.AddError("Unknown encryption intent", "Resolve kerberos_encryption_types before a mutation.")
	} else if config.KerberosEncryptionTypes.IsNull() && plan.KerberosEncryptionTypes.IsUnknown() {
		plan.KerberosEncryptionTypes = types.SetNull(types.StringType)
	}
	if config.AccountExpires.IsUnknown() {
		diagnostics.AddError("Unknown service account expiration", "Resolve account_expires before a mutation.")
	} else if config.AccountExpires.IsNull() && plan.AccountExpires.IsUnknown() {
		plan.AccountExpires = types.Int64Null()
	}
	if config.PasswordWOVersion.IsUnknown() {
		diagnostics.AddError("Unknown password version", "Resolve password_wo_version before a mutation.")
	} else if config.PasswordWOVersion.IsNull() && (plan.PasswordWOVersion.IsUnknown() || plan.PasswordWOVersion.IsNull()) {
		plan.PasswordWOVersion = types.Int64Value(0)
		if previous != nil {
			plan.PasswordWOVersion = previous.PasswordWOVersion
		}
	}
	for planned, configured := range map[*types.Set]*types.Set{
		&plan.ServicePrincipalNames: &config.ServicePrincipalNames, &plan.PasswordReaderSIDs: &config.PasswordReaderSIDs,
	} {
		if configured.IsUnknown() {
			diagnostics.AddError("Unknown service account collection", "Resolve configured SPNs and password reader SIDs before any mutation.")
		} else if configured.IsNull() && planned.IsUnknown() {
			*planned = types.SetNull(types.StringType)
		}
	}
	if config.ManagedPasswordIntervalDays.IsUnknown() {
		diagnostics.AddError("Unknown managed password interval", "Resolve the configured interval before any mutation.")
	} else if config.ManagedPasswordIntervalDays.IsNull() && plan.ManagedPasswordIntervalDays.IsUnknown() {
		plan.ManagedPasswordIntervalDays = types.Int64Null()
	}
	_, clearDiagnostics := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	return diagnostics
}

// clearProperties selects optional text attributes only, excluding identity, password, expiration and account/encryption masks.
func clearProperties(value types.Set) (map[string]bool, diag.Diagnostics) {
	result, allowed := map[string]bool{}, map[string]bool{}
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown service account clear intent", "Resolve clear_properties before a mutation.")
		return result, diagnostics
	}
	model, properties := ServiceAccountResourceModel{}, service_accounts.ServiceAccountProperties{}
	for _, field := range textProperties(&model, &properties) {
		if !strings.EqualFold(field.ldap, "sAMAccountName") {
			allowed[strings.ToLower(field.ldap)] = true
		}
	}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsNull() || text.IsUnknown() || !allowed[strings.ToLower(text.ValueString())] {
			diagnostics.AddError(
				"Invalid service account clear property",
				"Select an optional typed LDAP text attribute; identity, passwords and flag masks cannot be cleared here.",
			)
		} else {
			result[strings.ToLower(text.ValueString())] = true
		}
	}
	return result, diagnostics
}
