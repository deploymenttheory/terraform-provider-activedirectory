// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// objectSchema owns explicit raw intent, with directory visibility and binary observations exposed separately.
func objectSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable nonzero object GUID used for lifecycle and import.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":      schema.StringAttribute{Required: true, Description: "Literal object name. The SDK uses the selected class's naming attribute; changes rename the same GUID."},
		"parent_dn": schema.StringAttribute{Required: true, Description: "Explicit parent distinguished name. Changes move the same object within its domain."},
		"object_class": schema.StringAttribute{
			Required: true, Description: "Explicit schema LDAP display name for the instantiated class. Changing it replaces the object.",
			PlanModifiers: []planmodifier.String{classReplacement()},
		},
		"properties": schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: "Additional raw attributes to observe without owning their values."},
		"clear_attributes": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Explicit attributes to clear on an existing object. Omission never clears an attribute; unavailable readback alone never proves a clear.",
		},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"object_classes":     schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Ordered directory-observed class hierarchy."},
		"observed_attributes": schema.MapAttribute{
			Computed: true, ElementType: common.RawSetType(),
			Description: "Raw selected attribute values as base64 sets. Unavailable or partial values remain null, independently of configured intent.",
		},
		"attribute_status": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "present, absent, unloaded or partial for each selected raw attribute.",
		},
		"attribute_ranges": schema.MapAttribute{
			Computed:    true,
			ElementType: common.RawSetType(),
			Description: "Incomplete returned ranges, kept separate from complete attributes.",
		},
		"needs_update": schema.BoolAttribute{
			Computed:    true,
			Description: "A complete returned value set contradicted owned intent. Terraform schedules reconciliation; false does not prove hidden or absent values are unset.",
		},
		"recovery_dn": schema.StringAttribute{
			Computed:    true,
			Description: "Recovery locator if creation acceptance precedes usable GUID observation. Never an identity substitute.",
		},
	}
	for name, description := range map[string]string{
		"attributes":    "Owned attribute values as canonical base64 sets. Empty sets require clear_attributes; omitted attributes remain unowned.",
		"add_values":    "Selected base64 values to add while preserving other values. May initialize values during creation.",
		"remove_values": "Selected base64 values to remove on an existing object, preserving other values. The server enforces LDAP matching and strict errors.",
	} {
		attributes[name] = schema.MapAttribute{Optional: true, ElementType: common.RawSetType(), Description: description}
	}
	return schema.Schema{
		Description: "Manage one AD DS object by GUID with explicit class and raw ownership. Avoid competing typed/generic ownership of the same attributes. Deletion is leaf-only.",
		Attributes:  attributes,
	}
}

// classReplacement treats LDAP class names case-insensitively so casing edits never destroy the same schema instance.
func classReplacement() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = req.PlanValue.IsUnknown() || !strings.EqualFold(req.PlanValue.ValueString(), req.StateValue.ValueString())
	}, "Selecting a different instantiated class replaces the object; case-only edits preserve identity.",
		"Selecting a different instantiated class replaces the object; case-only edits preserve identity.")
}
