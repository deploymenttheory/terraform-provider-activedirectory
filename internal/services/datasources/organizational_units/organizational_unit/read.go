// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *OrganizationalUnitDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("OU client unavailable", "Configure the provider before reading organizational units.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.OrganizationalUnit(ctx, target, shared.ResolveIdentityOptions{Properties: []string{"ProtectedFromAccidentalDeletion"}})
	if result.Unit != nil && result.Unit.GUID != nil && target.Form() == shared.IdentityFormGUID && result.Unit.GUID.String() != target.String() {
		resp.Diagnostics.AddError("Unexpected OU lookup identity", "The directory returned a different object GUID from the requested identity.")
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.Unit)...)
	if err != nil || !result.Observation.Complete || result.Unit == nil {
		resp.Diagnostics.AddError(
			"OU lookup incomplete",
			"The directory did not return a complete usable OU observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
