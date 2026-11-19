// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// List performs one bounded SDK search when consumption begins and preserves usable identities before any later diagnostic.
func (r *ObjectListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	options, diagnostics := constructQuery(ctx, req)
	if r.client == nil {
		diagnostics.AddError("Object client unavailable", "Configure the provider before discovering generic objects.")
	}
	if diagnostics.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diagnostics)
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		if ctx.Err() != nil {
			push(diagnosticResult(ctx, req, "Object discovery canceled", "No directory query was submitted."))
			return
		}
		result, err := r.client.Search(ctx, options)
		slices.SortStableFunc(result.Objects, func(left, right shared.Object) int {
			leftGUID, leftValid := common.GUID(left)
			rightGUID, rightValid := common.GUID(right)
			if leftValid != rightValid {
				if leftValid {
					return -1
				}
				return 1
			}
			return strings.Compare(leftGUID, rightGUID)
		})
		seen := map[string]bool{}
		emitted := 0
		for _, object := range result.Objects {
			if ctx.Err() != nil {
				push(diagnosticResult(ctx, req, "Object discovery canceled", "Already emitted observations are retained; no further rows are emitted."))
				return
			}
			if emitted >= options.ResultLimit {
				break
			}
			values, projection := common.Project(object, options.Properties)
			if req.IncludeResource && !projection.HasError() && values["object_class"].IsNull() {
				projection.AddError("Object import snapshot incomplete", "The instantiated class was unavailable. A required resource field is never guessed.")
			}
			if projection.HasError() {
				row := req.NewListResult(ctx)
				row.Diagnostics.Append(projection...)
				if !push(row) {
					return
				}
				continue
			}
			id := values["id"].(types.String).ValueString()
			if seen[id] {
				if !push(diagnosticResult(ctx, req, "Duplicate object identity", "The search returned a repeated GUID. No duplicate import identity is emitted.")) {
					return
				}
				continue
			}
			seen[id] = true
			row := mapResult(ctx, req, values)
			row.Diagnostics.Append(projection...)
			if !push(row) {
				return
			}
			emitted++
		}
		if err != nil || (!result.Complete && !result.LimitReached) {
			push(diagnosticResult(ctx, req, "Object discovery incomplete", "The SDK search failed or did not complete. Usable identities are retained; peer text is withheld."))
		} else if result.LimitReached {
			row := list.ListResult{}
			row.Diagnostics.AddWarning("Object discovery reached its bound", "The caller's limit stopped materialization; the full search was not proven exhausted.")
			push(row)
		}
	}
}
