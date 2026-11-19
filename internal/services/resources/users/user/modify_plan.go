// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
)

// modifyUserPlan makes explicit clearing visible to Terraform and rejects a decreasing password version before apply.
func modifyUserPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config, previous UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
		if !plan.PasswordWOVersion.IsUnknown() && !plan.PasswordWOVersion.IsNull() && plan.PasswordWOVersion.ValueInt64() < previous.PasswordWOVersion.ValueInt64() {
			resp.Diagnostics.AddError("Password version cannot decrease", "Retain or increase password_wo_version. Version metadata cannot recover a prior password.")
		}
	}
	if plan.ClearProperties.IsUnknown() {
		return
	}
	clears, diagnostics := clearProperties(plan.ClearProperties)
	resp.Diagnostics.Append(diagnostics...)
	if req.State.Raw.IsNull() && len(clears) > 0 {
		resp.Diagnostics.AddError("User clear requires an existing object", "Create omits unconfigured properties. Explicit clears require an existing or imported user.")
	}
	ignored := users.UserProperties{}
	configured, planned := textProperties(&config, &ignored), textProperties(&plan, &ignored)
	for index, field := range planned {
		if !clears[strings.ToLower(field.ldap)] {
			continue
		}
		if !configured[index].value.IsNull() {
			resp.Diagnostics.AddError("Conflicting user property intent", "A property cannot be configured and included in clear_properties at the same time.")
		} else {
			*field.value = types.StringNull()
		}
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// preparePlan resolves only provider-generated unknowns for omitted computed properties; configured unknowns remain errors.
func preparePlan(plan, config *UserResourceModel, previous *UserResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	ignored := users.UserProperties{}
	configured, planned := textProperties(config, &ignored), textProperties(plan, &ignored)
	for index, field := range planned {
		if configured[index].value.IsUnknown() {
			diagnostics.AddError("Unknown user property", "Configured properties must be known before any user mutation.")
		} else if configured[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	configuredFlags, plannedFlags := boolProperties(config, &ignored), boolProperties(plan, &ignored)
	for index, field := range plannedFlags {
		if configuredFlags[index].value.IsUnknown() {
			diagnostics.AddError("Unknown user flag", "Configured account-control properties must be known before any mutation.")
		} else if configuredFlags[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.BoolNull()
		}
	}
	if config.AccountExpires.IsUnknown() {
		diagnostics.AddError("Unknown user expiration", "Resolve account_expires before any mutation.")
	} else if config.AccountExpires.IsNull() && plan.AccountExpires.IsUnknown() {
		plan.AccountExpires = types.Int64Null()
	}
	if config.ObjectClass.IsUnknown() || config.PasswordWOVersion.IsUnknown() {
		diagnostics.AddError("Unknown user creation or password intent", "Object class and password version must be known before any mutation.")
	}
	if config.ObjectClass.IsNull() && (plan.ObjectClass.IsUnknown() || plan.ObjectClass.IsNull()) {
		plan.ObjectClass = types.StringValue("user")
		if previous != nil {
			plan.ObjectClass = previous.ObjectClass
		}
	}
	if config.PasswordWOVersion.IsNull() && (plan.PasswordWOVersion.IsUnknown() || plan.PasswordWOVersion.IsNull()) {
		plan.PasswordWOVersion = types.Int64Value(0)
		if previous != nil {
			plan.PasswordWOVersion = previous.PasswordWOVersion
		}
	}
	_, clearDiagnostics := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	return diagnostics
}

// clearProperties accepts only explicitly owned optional text fields, never identity, expiration, flags or password material.
func clearProperties(value types.Set) (map[string]bool, diag.Diagnostics) {
	result, allowed := map[string]bool{}, map[string]bool{}
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown user clear intent", "clear_properties must be known before any mutation.")
		return result, diagnostics
	}
	model, properties := UserResourceModel{}, users.UserProperties{}
	for _, field := range textProperties(&model, &properties) {
		if field.terraform != "sam_account_name" {
			allowed[strings.ToLower(field.ldap)] = true
		}
	}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsNull() || text.IsUnknown() || !allowed[strings.ToLower(text.ValueString())] {
			diagnostics.AddError(
				"Invalid user clear property",
				"Select an optional typed LDAP text attribute; identity, expiration, password and account-control fields cannot be cleared here.",
			)
		} else {
			result[strings.ToLower(text.ValueString())] = true
		}
	}
	return result, diagnostics
}
