// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/groups"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Group client unavailable", "Configure the provider before reading groups.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.Group(ctx, target, shared.ResolveIdentityOptions{})
	if result.Group != nil && result.Group.GUID != nil && target.Form() == shared.IdentityFormGUID && result.Group.GUID.String() != target.String() {
		resp.Diagnostics.AddError("Unexpected group lookup identity", "The directory returned a different object GUID from the requested identity.")
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.Group)...)
	if err != nil || !result.Observation.Complete || result.Group == nil {
		resp.Diagnostics.AddError(
			"Group lookup incomplete",
			"The directory did not return a complete usable group observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
