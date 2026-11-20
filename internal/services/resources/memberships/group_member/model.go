// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import "github.com/hashicorp/terraform-plugin-framework/types"

// GroupMemberResourceModel identifies one direct link while keeping mutable directory names computed.
type GroupMemberResourceModel struct {
	ID         types.String `tfsdk:"id"`
	GroupGUID  types.String `tfsdk:"group_guid"`
	MemberGUID types.String `tfsdk:"member_guid"`
	GroupDN    types.String `tfsdk:"group_dn"`
	MemberDN   types.String `tfsdk:"member_dn"`
	IsMember   types.Bool   `tfsdk:"is_member"`
}
