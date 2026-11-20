// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// constructQuery resolves every configured bound before SDK I/O and never permits zero to select an unbounded default.
func constructQuery(model GroupMembersDataSourceModel) (shared.Identity, memberships.GroupMembersOptions, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var group shared.Identity
	guid, err := identity.CanonicalGUID(model.GroupGUID.ValueString())
	if err != nil || model.GroupGUID.IsNull() || model.GroupGUID.IsUnknown() {
		diagnostics.AddError("Invalid group membership scope", "group_guid must be a known nonzero group object GUID.")
	} else {
		group, _ = shared.IdentityByGUID(guid)
	}
	if model.Recursive.IsUnknown() {
		diagnostics.AddError("Unknown membership traversal mode", "Resolve recursive before reading group membership.")
	}
	options := memberships.GroupMembersOptions{Recursive: model.Recursive.ValueBool(), Properties: []string{"distinguishedName"}}
	for _, bound := range []struct {
		name               string
		value              types.Int64
		fallback, min, max int64
		target             *int
	}{
		{"max_objects", model.MaxObjects, 1000, 1, 10000, &options.Limits.MaxObjects},
		{"max_groups", model.MaxGroups, 100, 1, 1000, &options.Limits.MaxGroups},
		{"max_ranges", model.MaxRanges, 100, 1, 1000, &options.Limits.MaxRanges},
		{"max_depth", model.MaxDepth, 8, 1, 32, &options.Limits.MaxDepth},
		{"max_bytes", model.MaxBytes, 8 << 20, 1024, 64 << 20, &options.Limits.MaxBytes},
	} {
		value := bound.fallback
		if bound.value.IsUnknown() {
			diagnostics.AddError("Unknown membership traversal bound", "Resolve "+bound.name+" before reading group membership.")
			continue
		}
		if !bound.value.IsNull() {
			value = bound.value.ValueInt64()
		}
		if value < bound.min || value > bound.max {
			diagnostics.AddError("Invalid membership traversal bound", bound.name+" is outside its documented finite range. Zero never enables unbounded retrieval.")
			continue
		}
		*bound.target = int(value)
	}
	return group, options, diagnostics
}
