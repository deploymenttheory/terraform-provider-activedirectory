// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// groupSchema expresses property ownership without exposing membership or writable groupType masks.
func groupSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable group object GUID used for import and lifecycle identity.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":             schema.StringAttribute{Required: true, Description: "Literal group name. Changes rename the same object GUID."},
		"parent_dn":        schema.StringAttribute{Required: true, Description: "Explicit parent distinguished name. Changes move the same group within its domain."},
		"sam_account_name": schema.StringAttribute{Required: true, Description: "Explicit sAMAccountName. No account name or container is guessed."},
		"category": schema.StringAttribute{
			Required: true, Description: "security or distribution. Updates use the SDK's typed patch and preserve unrelated groupType bits.",
		},
		"scope": schema.StringAttribute{
			Required: true, Description: "global, domain_local or universal. The directory decides whether a requested scope conversion is permitted.",
		},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"sid":                schema.StringAttribute{Computed: true, Description: "Directory-assigned security identifier when visible."},
		"recovery_dn":        schema.StringAttribute{Computed: true, Description: "Recovery locator when creation acceptance precedes GUID observation."},
		"clear_properties": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Explicit description, displayName, wWWHomePage or managedBy attributes to clear on an existing group. Omission retains observed values.",
		},
	}
	for name, ldap := range map[string]string{
		"description": "description", "display_name": "displayName", "home_page": "wWWHomePage", "managed_by_dn": "managedBy",
	} {
		attributes[name] = schema.StringAttribute{
			Optional: true, Computed: true,
			Description:   "Typed " + ldap + " property. Omit to retain the observed value; clear_properties requests an explicit clear.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	return schema.Schema{
		Description: "Manage one AD DS group by GUID. Category/scope changes preserve unrelated flags. Membership belongs to separate edge resources; deletion removes the group only.",
		Attributes:  attributes,
	}
}
