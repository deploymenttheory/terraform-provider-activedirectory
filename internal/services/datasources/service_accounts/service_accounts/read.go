// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_accounts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// Read executes one bounded SDK query and diagnoses incomplete retrieval without discarding usable rows.
func (d *ServiceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, options, diagnostics := constructQuery(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Service account client unavailable", "Configure the provider before discovering service accounts.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.ServiceAccounts(ctx, options)
	if err != nil {
		result.Observation.Complete = false
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, options, result)...)
	if err != nil || (!result.Observation.Complete && !result.Observation.LimitReached) {
		resp.Diagnostics.AddError(
			"Service account discovery incomplete",
			"The SDK returned an incomplete or failed search. Usable rows are retained; no claim of complete absence is made. Peer text is withheld.",
		)
	} else if result.Observation.LimitReached {
		resp.Diagnostics.AddWarning(
			"Service account discovery reached its bound",
			"The configured result_limit stopped retrieval. complete is false; additional matches may or may not exist.",
		)
	}
}
