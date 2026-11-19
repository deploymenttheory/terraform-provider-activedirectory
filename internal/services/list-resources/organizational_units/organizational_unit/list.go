// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

// List performs one bounded SDK query when iteration starts, then respects cancellation and consumer stop.
func (r *OrganizationalUnitListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	options, diagnostics := constructQuery(ctx, req)
	if r.client == nil {
		diagnostics.AddError("OU client unavailable", "Configure the provider before discovering organizational units.")
	}
	if diagnostics.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diagnostics)
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		if ctx.Err() != nil {
			push(diagnosticResult(ctx, req, "OU discovery canceled", "No directory query was performed."))
			return
		}
		result, err := r.client.OrganizationalUnits(ctx, options)
		slices.SortStableFunc(result.Units, func(left, right organizational_units.OrganizationalUnit) int {
			if left.GUID == nil && right.GUID == nil {
				return 0
			}
			if left.GUID == nil {
				return 1
			}
			if right.GUID == nil {
				return -1
			}
			return strings.Compare(left.GUID.String(), right.GUID.String())
		})
		seen := map[string]bool{}
		emitted := 0
		for _, unit := range result.Units {
			if ctx.Err() != nil {
				push(diagnosticResult(ctx, req, "OU discovery canceled", "Already returned observations are retained; no further rows are emitted."))
				return
			}
			if emitted >= options.ResultLimit {
				break
			}
			values, projection := common.Project(unit)
			if projection.HasError() {
				row := req.NewListResult(ctx)
				row.Diagnostics.Append(projection...)
				if !push(row) {
					return
				}
				continue
			}
			id := unit.GUID.String()
			if seen[id] {
				if !push(diagnosticResult(ctx, req, "Duplicate OU identity", "The query returned a duplicate GUID. No duplicate import identity is emitted.")) {
					return
				}
				continue
			}
			seen[id] = true
			if !push(mapResult(ctx, req, values)) {
				return
			}
			emitted++
		}
		if err != nil || (!result.Observation.Complete && !result.Observation.LimitReached) {
			push(diagnosticResult(ctx, req, "OU discovery incomplete", "The SDK search failed or did not complete. Usable identities are retained; peer error text is withheld."))
		} else if result.Observation.LimitReached {
			row := list.ListResult{}
			row.Diagnostics.AddWarning("OU discovery reached its bound", "The selected limit stopped materialization; this result does not prove the full search was exhausted.")
			push(row)
		}
	}
}
