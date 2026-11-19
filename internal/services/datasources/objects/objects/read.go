// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// Read executes one bounded SDK query and diagnoses incomplete retrieval without discarding usable rows.
func (d *ObjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	model, options, diagnostics := constructQuery(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Object client unavailable", "Configure the provider before discovering objects.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := d.client.Search(ctx, options)
	if err != nil {
		result.Complete = false
	}
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, options, result)...)
	if err != nil || (!result.Complete && !result.LimitReached) {
		resp.Diagnostics.AddError(
			"Object discovery incomplete",
			"The SDK returned an incomplete or failed search. Usable rows are retained; no claim of complete absence is made. Peer text is withheld.",
		)
	} else if result.LimitReached {
		resp.Diagnostics.AddWarning(
			"Object discovery reached its bound",
			"The configured result_limit stopped retrieval. complete is false; additional matches may or may not exist.",
		)
	}
}
