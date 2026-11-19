// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// Read emits state only for a positively confirmed direct link, without exposing peer diagnostic text.
func (d *GroupMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, group, member, diagnostics := constructLookup(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Membership client unavailable", "Configure the provider before reading direct links.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	edge, err := common.Read(ctx, d.client, group, member)
	if err != nil || !edge.Confirmed || !edge.Present {
		resp.Diagnostics.AddError(
			"Direct membership unavailable",
			"The exact direct link could not be confirmed. Primary or effective membership is insufficient; peer error text is withheld.",
		)
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, edge)...)
}
