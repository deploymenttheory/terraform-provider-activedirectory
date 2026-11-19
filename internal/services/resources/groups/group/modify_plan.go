// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
)

// modifyGroupPlan exposes explicit clearing to Terraform without treating omitted optional properties as owned emptiness.
func modifyGroupPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || plan.ClearProperties.IsUnknown() {
		return
	}
	clears, diagnostics := clearProperties(plan.ClearProperties)
	resp.Diagnostics.Append(diagnostics...)
	if req.State.Raw.IsNull() && len(clears) > 0 {
		resp.Diagnostics.AddError("Group clear requires an existing object", "Explicit property clears apply to an existing or imported group.")
	}
	ignored := groups.UpdateGroupRequest{}
	planned, configured := propertyBindings(&plan, &ignored), propertyBindings(&config, &ignored)
	for index, field := range planned {
		if !clears[strings.ToLower(field.ldap)] {
			continue
		}
		if !configured[index].value.IsNull() {
			resp.Diagnostics.AddError("Conflicting group property intent", "A property cannot be configured and included in clear_properties at the same time.")
		} else {
			*field.value = types.StringNull()
		}
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// preparePlan resolves only unknowns generated for omitted computed fields; unresolved configured values remain refusals.
func preparePlan(plan, config *GroupResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	ignored := groups.UpdateGroupRequest{}
	planned, configured := propertyBindings(plan, &ignored), propertyBindings(config, &ignored)
	for index, field := range planned {
		if configured[index].value.IsUnknown() {
			diagnostics.AddError("Unknown group property", "Resolve configured properties before submitting any group mutation.")
		} else if configured[index].value.IsNull() && field.value.IsUnknown() {
			*field.value = types.StringNull()
		}
	}
	return diagnostics
}
