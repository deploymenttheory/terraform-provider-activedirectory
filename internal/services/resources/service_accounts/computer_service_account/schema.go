// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func computerServiceAccountSchema() schema.Schema {
	return schema.Schema{
		Description: "Manage one direct computer/account link. Existing links are adopted; deletion removes only this link. " +
			"Other accounts and both directory objects are preserved. No local installation, password retrieval or password-reader grants are performed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Canonical computer GUID/account GUID composite identity."},
			"computer_guid": schema.StringAttribute{
				Required: true, Description: "Existing computer object GUID. Changing either endpoint replaces the owned link.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_account_guid": schema.StringAttribute{
				Required: true, Description: "Existing standalone or group managed service-account object GUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"computer_dn":        schema.StringAttribute{Computed: true, Description: "Current observed computer distinguished name. Renames preserve link identity."},
			"service_account_dn": schema.StringAttribute{Computed: true, Description: "Current observed account distinguished name. Renames preserve link identity."},
			"is_associated": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the direct link was positively confirmed. Null means current visibility is indeterminate; an explicit apply ensures the configured link.",
			},
		},
	}
}
