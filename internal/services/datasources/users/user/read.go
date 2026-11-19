// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/users"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("User client unavailable", "Configure the provider before reading users.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.User(ctx, target, shared.ResolveIdentityOptions{})
	if result.User != nil && result.User.GUID != nil && target.Form() == shared.IdentityFormGUID && result.User.GUID.String() != target.String() {
		resp.Diagnostics.AddError("Unexpected user lookup identity", "The directory returned a different object GUID from the requested identity.")
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.User)...)
	if err != nil || !result.Observation.Complete || result.User == nil {
		resp.Diagnostics.AddError(
			"User lookup incomplete",
			"The directory did not return a complete usable user observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
