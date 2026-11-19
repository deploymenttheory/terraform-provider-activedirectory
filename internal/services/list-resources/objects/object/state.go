// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
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
		for _, name := range []string{"attributes", "add_values", "remove_values"} {
			result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root(name), types.MapNull(common.RawSetType()))...)
		}
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("clear_attributes"), types.SetNull(types.StringType))...)
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("needs_update"), types.BoolValue(false))...)
		var properties types.List
		result.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("properties"), &properties)...)
		selected := types.SetNull(types.StringType)
		if !properties.IsNull() {
			selected = types.SetValueMust(types.StringType, properties.Elements())
		}
		result.Diagnostics.Append(result.Resource.SetAttribute(ctx, path.Root("properties"), selected)...)
	}
	return result
}

// diagnosticResult preserves retrieval failure after any usable identities that were already observed.
func diagnosticResult(ctx context.Context, req list.ListRequest, summary, detail string) list.ListResult {
	result := req.NewListResult(ctx)
	result.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic(summary, detail)}
	return result
}
