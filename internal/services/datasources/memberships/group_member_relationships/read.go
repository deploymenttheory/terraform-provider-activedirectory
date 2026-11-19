// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member_relationships

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// Read retains usable direct-link rows while reporting failed, incomplete or bounded discovery explicitly.
func (d *GroupMemberRelationshipsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, group, limit, diagnostics := constructQuery(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Membership client unavailable", "Configure the provider before discovering group links.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := common.Discover(ctx, d.client, group, limit)
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, observed)...)
	if err != nil && !observed.Bounded {
		resp.Diagnostics.AddError(
			"Membership discovery incomplete",
			"Returned usable link identities are retained. The directory query or member resolution did not complete; peer error text is withheld.",
		)
	} else if observed.Bounded {
		resp.Diagnostics.AddWarning("Membership discovery reached its bound", "complete is false; returned links do not prove the whole group was enumerated.")
	}
}
