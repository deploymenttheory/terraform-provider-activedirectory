// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
)

// modifyComputerPlan exposes selected clears and prevents decreasing password intent before apply.
func modifyComputerPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config, previous ComputerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
		if !plan.PasswordWOVersion.IsUnknown() && !plan.PasswordWOVersion.IsNull() && plan.PasswordWOVersion.ValueInt64() < previous.PasswordWOVersion.ValueInt64() {
			resp.Diagnostics.AddError("Password version cannot decrease", "Retain or increase password_wo_version; metadata cannot recover an old password.")
		}
	}
	if plan.ClearProperties.IsUnknown() {
		return
	}
	clears, diagnostics := clearProperties(plan.ClearProperties)
	resp.Diagnostics.Append(diagnostics...)
	if req.State.Raw.IsNull() && len(clears) > 0 {
		resp.Diagnostics.AddError("Computer clear requires an existing object", "Explicit property clearing applies only to an existing or imported computer.")
	}
	ignored := computers.ComputerProperties{}
	planned, configured := textProperties(&plan, &ignored), textProperties(&config, &ignored)
	for index, field := range planned {
		if !clears[strings.ToLower(field.ldap)] {
			continue
		}
		if !configured[index].value.IsNull() {
			resp.Diagnostics.AddError("Conflicting computer property intent", "A property cannot have a configured value and be explicitly cleared at the same time.")
		} else {
			*field.value = types.StringNull()
		}
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// preparePlan resolves provider-generated unknowns only; unknown configured inputs never authorize mutation.
func preparePlan(plan, config *ComputerResourceModel, previous *ComputerResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	properties, control := computers.ComputerProperties{}, accounts.AccountControlPatch{}
	planned, configured := textProperties(plan, &properties), textProperties(config, &properties)
	for index, field := range planned {
		if configured[index].value.IsUnknown() {
			diagnostics.AddError("Unknown computer property", "Resolve configured properties before a mutation.")
		} else if configured[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	plannedFlags, configuredFlags := boolProperties(plan, &control), boolProperties(config, &control)
	for index, field := range plannedFlags {
		if configuredFlags[index].value.IsUnknown() {
			diagnostics.AddError("Unknown computer account setting", "Resolve configured account settings before a mutation.")
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
		diagnostics.AddError("Unknown computer expiration", "Resolve account_expires before a mutation.")
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
	_, clearDiagnostics := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	return diagnostics
}

// clearProperties selects optional text attributes only, excluding identity, password, expiration and account/encryption masks.
func clearProperties(value types.Set) (map[string]bool, diag.Diagnostics) {
	result, allowed := map[string]bool{}, map[string]bool{}
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown computer clear intent", "Resolve clear_properties before a mutation.")
		return result, diagnostics
	}
	model, properties := ComputerResourceModel{}, computers.ComputerProperties{}
	for _, field := range textProperties(&model, &properties) {
		if !strings.EqualFold(field.ldap, "sAMAccountName") {
			allowed[strings.ToLower(field.ldap)] = true
		}
	}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsNull() || text.IsUnknown() || !allowed[strings.ToLower(text.ValueString())] {
			diagnostics.AddError("Invalid computer clear property", "Select an optional typed LDAP text attribute; identity, passwords and flag masks cannot be cleared here.")
		} else {
			result[strings.ToLower(text.ValueString())] = true
		}
	}
	return result, diagnostics
}
