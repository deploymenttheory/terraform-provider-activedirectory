// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// constructLookup validates both inputs before reading directory objects or evaluating a link assertion.
func constructLookup(ctx context.Context, config tfsdk.Config) (GroupMemberDataSourceModel, shared.Identity, shared.Identity, diag.Diagnostics) {
	var model GroupMemberDataSourceModel
	diagnostics := config.GetAttribute(ctx, path.Root("group_guid"), &model.GroupGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("member_guid"), &model.MemberGUID)...)
	group, member, validation := common.ConstructPair(model.GroupGUID, model.MemberGUID)
	diagnostics.Append(validation...)
	return model, group, member, diagnostics
}
