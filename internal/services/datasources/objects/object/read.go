// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *ObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	properties, diagnostics := common.ConstructProperties(model.Properties)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Object client unavailable", "Configure the provider before reading objects.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{Properties: properties})
	if result.Object != nil && target.Form() == shared.IdentityFormGUID {
		if returned, valid := common.GUID(*result.Object); valid && returned != target.String() {
			resp.Diagnostics.AddError("Unexpected object lookup identity", "The directory returned a different object GUID from the requested identity.")
			return
		}
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.Object, properties)...)
	if err != nil || !result.Complete || result.Object == nil {
		resp.Diagnostics.AddError(
			"Object lookup incomplete",
			"The directory did not return a complete usable object observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
