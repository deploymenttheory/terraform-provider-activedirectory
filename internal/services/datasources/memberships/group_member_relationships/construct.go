// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member_relationships

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// constructQuery validates an explicit group and bounded member collection before directory I/O.
func constructQuery(ctx context.Context, config tfsdk.Config) (common.QueryModel, shared.Identity, int, diag.Diagnostics) {
	var model common.QueryModel
	diagnostics := config.GetAttribute(ctx, path.Root("group_guid"), &model.GroupGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("result_limit"), &model.ResultLimit)...)
	group, limit, validation := common.ConstructQuery(model)
	diagnostics.Append(validation...)
	return model, group, limit, diagnostics
}
