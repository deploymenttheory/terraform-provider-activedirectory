// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func groupMemberSchema() schema.Schema {
	return schema.Schema{
		Description: "Manage one direct group/member link. Existing links are adopted; deletion removes only this link. " +
			"Other members and both directory objects are preserved. Primary and nested membership are not direct links.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Canonical group GUID/member GUID composite identity."},
			"group_guid": schema.StringAttribute{
				Required: true, Description: "Existing group object GUID. Changing either endpoint replaces the owned link.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"member_guid": schema.StringAttribute{
				Required: true, Description: "Existing user, group, computer or service-account object GUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_dn":  schema.StringAttribute{Computed: true, Description: "Current observed group distinguished name. Renames preserve link identity."},
			"member_dn": schema.StringAttribute{Computed: true, Description: "Current observed member distinguished name. Renames preserve link identity."},
			"is_member": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the direct link was positively confirmed. Null means current visibility is indeterminate; an explicit apply ensures the configured link.",
			},
		},
	}
}
