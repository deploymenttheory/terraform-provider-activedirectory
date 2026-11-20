// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import "github.com/hashicorp/terraform-plugin-framework/types"

// GroupMembersDataSourceModel separates configured scope and aggregate traversal budgets from observed results.
type GroupMembersDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	GroupGUID       types.String `tfsdk:"group_guid"`
	GroupDN         types.String `tfsdk:"group_dn"`
	Recursive       types.Bool   `tfsdk:"recursive"`
	MaxObjects      types.Int64  `tfsdk:"max_objects"`
	MaxGroups       types.Int64  `tfsdk:"max_groups"`
	MaxRanges       types.Int64  `tfsdk:"max_ranges"`
	MaxDepth        types.Int64  `tfsdk:"max_depth"`
	MaxBytes        types.Int64  `tfsdk:"max_bytes"`
	Members         types.List   `tfsdk:"members"`
	Unresolved      types.List   `tfsdk:"unresolved"`
	Complete        types.Bool   `tfsdk:"complete"`
	LimitReached    types.Bool   `tfsdk:"limit_reached"`
	GroupsVisited   types.Int64  `tfsdk:"groups_visited"`
	VisibilityKnown types.Bool   `tfsdk:"visibility_known"`
}
