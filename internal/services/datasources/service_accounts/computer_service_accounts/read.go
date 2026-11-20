// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_accounts

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// Read retains usable direct-link rows while reporting failed, incomplete or bounded discovery explicitly.
func (d *ComputerServiceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, computer, limit, diagnostics := constructQuery(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Association client unavailable", "Configure the provider before discovering computer links.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := common.Discover(ctx, d.client, computer, limit)
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, observed)...)
	switch {
	case errors.Is(err, common.ErrVisibility):
		resp.Diagnostics.AddWarning(
			"Association visibility is indeterminate",
			"No direct values were returned, which cannot distinguish an empty computer from hidden association. complete remains false; no absence is inferred.",
		)
	case err != nil && !observed.Bounded:
		resp.Diagnostics.AddError(
			"Association discovery incomplete",
			"Returned usable link identities are retained. The directory query or account resolution did not complete; peer error text is withheld.",
		)
	case observed.Bounded:
		resp.Diagnostics.AddWarning("Association discovery reached its bound", "complete is false; returned links do not prove the whole computer was enumerated.")
	}
}
