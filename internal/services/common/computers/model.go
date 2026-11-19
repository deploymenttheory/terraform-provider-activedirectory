// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// SearchConfig reads the shared explicit search fields from a data-source or list-resource configuration.
func SearchConfig(ctx context.Context, config tfsdk.Config) (SearchModel, diag.Diagnostics) {
	var model SearchModel
	diagnostics := config.GetAttribute(ctx, path.Root("base_dn"), &model.BaseDN)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("ldap_filter"), &model.LDAPFilter)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("scope"), &model.Scope)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("page_size"), &model.PageSize)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("result_limit"), &model.ResultLimit)...)
	return model, diagnostics
}
