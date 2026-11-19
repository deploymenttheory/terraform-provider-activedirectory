// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapResult emits only the managed resource's exact GUID identity schema and usable directory observations.
func mapResult(ctx context.Context, req list.ListRequest, values map[string]attr.Value) list.ListResult {
	result := req.NewListResult(ctx)
	result.DisplayName = values["distinguished_name"].(types.String).ValueString()
	result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root("id"), values["id"])...)
	if req.IncludeResource {
		for name, value := range values {
			result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root(name), value)...)
		}
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("recovery_dn"), types.StringNull())...)
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("clear_properties"), types.SetNull(types.StringType))...)
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("password_wo"), types.StringNull())...)
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("password_wo_version"), types.Int64Value(0))...)
	}
	return result
}

// diagnosticResult preserves retrieval failure after any usable identities that were already observed.
func diagnosticResult(ctx context.Context, req list.ListRequest, summary, detail string) list.ListResult {
	result := req.NewListResult(ctx)
	result.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic(summary, detail)}
	return result
}
