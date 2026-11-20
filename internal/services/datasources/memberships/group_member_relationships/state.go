// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member_relationships

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// mapState preserves configured query inputs and materializes only GUID-backed direct relationship rows.
func mapState(ctx context.Context, state *tfsdk.State, model common.QueryModel, observed common.Discovery) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	rows := []attr.Value{}
	for _, edge := range observed.Edges {
		values, err := edge.Values()
		if err != nil {
			diagnostics.AddError("Incomplete direct-link identity", "A discovered row cannot become a usable import identity.")
			continue
		}
		row, projection := types.ObjectValue(common.AttributeTypes(), values)
		diagnostics.Append(projection...)
		rows = append(rows, row)
	}
	members, projection := types.ListValue(types.ObjectType{AttrTypes: common.AttributeTypes()}, rows)
	diagnostics.Append(projection...)
	id, groupDN := types.StringNull(), types.StringNull()
	if observed.Group.DN != "" {
		id, groupDN = types.StringValue(observed.Group.GUID.String()), types.StringValue(observed.Group.DN)
	}
	for name, value := range map[string]attr.Value{
		"id": id, "group_guid": model.GroupGUID, "group_dn": groupDN, "result_limit": model.ResultLimit,
		"members": members, "complete": types.BoolValue(observed.Complete), "limit_reached": types.BoolValue(observed.Bounded),
	} {
		diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diagnostics
}
