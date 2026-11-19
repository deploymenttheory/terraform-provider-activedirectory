// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_accounts"
)

// Read resolves one explicit identity and retains usable partial observations without claiming completeness.
func (d *ServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	target, diagnostics := common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Service account client unavailable", "Configure the provider before reading service accounts.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.ServiceAccount(ctx, target, shared.ResolveIdentityOptions{})
	if result.Account != nil && result.Account.GUID != nil && target.Form() == shared.IdentityFormGUID && result.Account.GUID.String() != target.String() {
		resp.Diagnostics.AddError("Unexpected service_account lookup identity", "The directory returned a different object GUID from the requested identity.")
		return
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, result.Account)...)
	if err != nil || !result.Observation.Complete || result.Account == nil {
		resp.Diagnostics.AddError(
			"Service account lookup incomplete",
			"The directory did not return a complete usable service_account observation. The object may be missing or inaccessible; peer error text is withheld.",
		)
	}
}
