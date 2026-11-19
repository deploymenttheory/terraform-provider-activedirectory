// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_accounts"
)

// mapState preserves explicit lookup inputs and only usable SDK observations, including partial read results.
func mapState(ctx context.Context, state *tfsdk.State, model ServiceAccountDataSourceModel, unit *service_accounts.ServiceAccount) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if unit != nil {
		values, projection := common.Project(*unit)
		diagnostics.Append(projection...)
		for name, value := range values {
			diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
		}
	}
	diagnostics.Append(state.SetAttribute(ctx, path.Root("object_guid"), model.ObjectGUID)...)
	diagnostics.Append(state.SetAttribute(ctx, path.Root("dn"), model.DN)...)
	return diagnostics
}
