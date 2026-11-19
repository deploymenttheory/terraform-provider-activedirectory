// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import "github.com/hashicorp/terraform-plugin-framework/types"

// GroupMemberDataSourceModel selects one ordered GUID pair without owning the link.
type GroupMemberDataSourceModel struct {
	GroupGUID  types.String `tfsdk:"group_guid"`
	MemberGUID types.String `tfsdk:"member_guid"`
}
