// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/computers"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *ComputerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Computer client unavailable", "Configure the provider before reading computers.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.Computer(ctx, target, shared.ResolveIdentityOptions{})
	if result.Computer != nil && result.Computer.GUID != nil && target.Form() == shared.IdentityFormGUID && result.Computer.GUID.String() != target.String() {
		resp.Diagnostics.AddError("Unexpected computer lookup identity", "The directory returned a different object GUID from the requested identity.")
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.Computer)...)
	if err != nil || !result.Observation.Complete || result.Computer == nil {
		resp.Diagnostics.AddError(
			"Computer lookup incomplete",
			"The directory did not return a complete usable computer observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
