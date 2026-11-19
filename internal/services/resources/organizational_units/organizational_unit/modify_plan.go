// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
)

// ModifyPlan distinguishes an explicit attribute clear from simply omitting optional configuration.
func (*OrganizationalUnitResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config, plan OrganizationalUnitResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.ClearProperties.IsUnknown() {
		return
	}
	clears, diagnostics := clearProperties(plan.ClearProperties)
	resp.Diagnostics.Append(diagnostics...)
	if req.State.Raw.IsNull() && len(clears) > 0 {
		resp.Diagnostics.AddError("OU clear requires an existing object", "Create omits unconfigured attributes. Apply clear_properties only to an existing or imported OU.")
	}
	ignored := organizational_units.OUProperties{}
	configured, planned := propertyBindings(&config, &ignored), propertyBindings(&plan, &ignored)
	for index, field := range planned {
		if !clears[strings.ToLower(field.ldap)] {
			continue
		}
		if !configured[index].value.IsNull() {
			resp.Diagnostics.AddError("Conflicting OU property intent", "An attribute cannot be configured and included in clear_properties at the same time.")
			continue
		}
		*field.value = types.StringNull()
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// preparePlan removes only provider-generated unknowns for omitted computed fields, never unknown user input.
func preparePlan(plan, config *OrganizationalUnitResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	ignored := organizational_units.OUProperties{}
	planned, configured := propertyBindings(plan, &ignored), propertyBindings(config, &ignored)
	for index, field := range planned {
		if configured[index].value.IsUnknown() {
			diagnostics.AddError("Unknown OU property", "Configured properties must be known before any phase of a directory mutation.")
		} else if configured[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	_, clearDiagnostics := clearProperties(plan.ClearProperties)
	diagnostics.Append(clearDiagnostics...)
	return diagnostics
}

func clearProperties(value types.Set) (map[string]bool, diag.Diagnostics) {
	result := map[string]bool{}
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddError("Unknown OU clear intent", "clear_properties must be known before a directory mutation.")
		return result, diagnostics
	}
	allowed := map[string]bool{"description": true, "displayname": true, "l": true, "c": true, "st": true, "street": true, "postalcode": true, "managedby": true}
	for _, element := range value.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsUnknown() || text.IsNull() || !allowed[strings.ToLower(text.ValueString())] {
			diagnostics.AddError(
				"Invalid OU clear property",
				"Select a supported optional LDAP attribute; identity, location, protection and arbitrary attributes cannot be cleared here.",
			)
			continue
		}
		result[strings.ToLower(text.ValueString())] = true
	}
	return result, diagnostics
}
