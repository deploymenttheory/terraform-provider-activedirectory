// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/list"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// List materializes bounded direct links only when consumed, then respects cancellation and consumer stop.
func (r *ComputerServiceAccountListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	computer, limit, diagnostics := constructQuery(ctx, req)
	if r.client == nil {
		diagnostics.AddError("Association client unavailable", "Configure the provider before discovering direct links.")
	}
	if diagnostics.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diagnostics)
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		if ctx.Err() != nil {
			push(diagnosticResult(ctx, req, "Association discovery canceled", "No directory query was performed."))
			return
		}
		observed, err := common.Discover(ctx, r.client, computer, limit)
		for _, edge := range observed.Edges {
			if ctx.Err() != nil {
				push(diagnosticResult(ctx, req, "Association discovery canceled", "Returned identities are retained; no more rows are emitted."))
				return
			}
			if !push(mapResult(ctx, req, edge)) {
				return
			}
		}
		switch {
		case errors.Is(err, common.ErrVisibility):
			row := req.NewListResult(ctx)
			row.Diagnostics.AddWarning(
				"Association visibility is indeterminate",
				"An omitted account attribute cannot establish an empty computer. No complete absence is claimed.",
			)
			push(row)
		case err != nil && !observed.Bounded:
			push(
				diagnosticResult(
					ctx,
					req,
					"Association discovery incomplete",
					"Usable link identities are retained. Range traversal or account resolution failed; peer error text is withheld.",
				),
			)
		case observed.Bounded:
			row := req.NewListResult(ctx)
			row.Diagnostics.AddWarning("Association discovery reached its bound", "Returned links do not prove the whole computer was enumerated.")
			push(row)
		}
	}
}
