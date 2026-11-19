// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// constructPair validates the whole operation and refuses disagreement with an already established composite identity.
func constructPair(model GroupMemberResourceModel) (shared.Identity, shared.Identity, string, diag.Diagnostics) {
	group, member, diagnostics := common.ConstructPair(model.GroupGUID, model.MemberGUID)
	if diagnostics.HasError() {
		return group, member, "", diagnostics
	}
	id, _ := identity.RelationshipID(group.String(), member.String())
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != id {
		diagnostics.AddError("Conflicting membership identity", "The stored composite ID disagrees with the endpoint GUIDs. No directory operation was performed.")
	}
	return group, member, id, diagnostics
}

// constructAdd submits exactly one persistent direct link with native permissive duplicate-add semantics.
func constructAdd(group, member shared.Identity) memberships.AddGroupMembersRequest {
	return memberships.AddGroupMembersRequest{Identity: group, Members: []shared.Identity{member}, Mode: memberships.GroupMembershipPermissive}
}

// constructRemove removes exactly the owned link with native permissive missing-link semantics.
func constructRemove(group, member shared.Identity) memberships.RemoveGroupMembersRequest {
	return memberships.RemoveGroupMembersRequest{Identity: group, Members: []shared.Identity{member}, Mode: memberships.GroupMembershipPermissive}
}
