// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// List materializes bounded direct links only when consumed, then respects cancellation and consumer stop.
func (r *GroupMemberListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	group, limit, diagnostics := constructQuery(ctx, req)
	if r.client == nil {
		diagnostics.AddError("Membership client unavailable", "Configure the provider before discovering direct links.")
	}
	if diagnostics.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diagnostics)
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		if ctx.Err() != nil {
			push(diagnosticResult(ctx, req, "Membership discovery canceled", "No directory query was performed."))
			return
		}
		observed, err := common.Discover(ctx, r.client, group, limit)
		for _, edge := range observed.Edges {
			if ctx.Err() != nil {
				push(diagnosticResult(ctx, req, "Membership discovery canceled", "Returned identities are retained; no more rows are emitted."))
				return
			}
			if !push(mapResult(ctx, req, edge)) {
				return
			}
		}
		if err != nil && !observed.Bounded {
			push(
				diagnosticResult(
					ctx,
					req,
					"Membership discovery incomplete",
					"Usable link identities are retained. Range traversal or member resolution failed; peer error text is withheld.",
				),
			)
		} else if observed.Bounded {
			row := req.NewListResult(ctx)
			row.Diagnostics.AddWarning("Membership discovery reached its bound", "Returned links do not prove the whole group was enumerated.")
			push(row)
		}
	}
}
