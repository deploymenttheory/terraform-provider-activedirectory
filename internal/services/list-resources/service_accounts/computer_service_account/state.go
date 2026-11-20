// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// mapResult emits the managed resource's exact composite identity and optional resource snapshot.
func mapResult(ctx context.Context, req list.ListRequest, edge common.Edge) list.ListResult {
	result := req.NewListResult(ctx)
	values, err := edge.Values()
	if err != nil {
		result.Diagnostics.AddError("Incomplete association identity", "No usable direct-link import identity can be emitted.")
		return result
	}
	result.DisplayName = edge.Computer.DN + " -> " + edge.Account.DN
	result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root("id"), values["id"].(types.String))...)
	if req.IncludeResource {
		for name, value := range values {
			result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root(name), value)...)
		}
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("is_associated"), types.BoolValue(true))...)
	}
	return result
}

// diagnosticResult preserves a failed query after any usable rows without including raw directory diagnostics.
func diagnosticResult(ctx context.Context, req list.ListRequest, summary, detail string) list.ListResult {
	result := req.NewListResult(ctx)
	result.Diagnostics.AddError(summary, detail)
	return result
}
