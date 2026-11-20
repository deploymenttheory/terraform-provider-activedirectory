// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// mapState preserves configured GUID spelling while computing canonical relationship identity and observed names.
func mapState(model *GroupMemberResourceModel, edge common.Edge) {
	id, _ := identity.RelationshipID(edge.Group.GUID.String(), edge.Member.GUID.String())
	model.ID = types.StringValue(id)
	model.GroupDN, model.MemberDN = types.StringValue(edge.Group.DN), types.StringValue(edge.Member.DN)
	model.IsMember = types.BoolNull()
	if edge.Confirmed {
		model.IsMember = types.BoolValue(edge.Present)
	}
}

// persistState retains the stable pair in both Terraform state and resource identity after accepted or uncertain effects.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model GroupMemberResourceModel) diag.Diagnostics {
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() && !model.ID.IsUnknown() {
		diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), model.ID)...)
	}
	return diagnostics
}
