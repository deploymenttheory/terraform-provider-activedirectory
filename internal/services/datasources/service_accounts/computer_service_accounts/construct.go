// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_accounts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// constructQuery validates an explicit computer and bounded account collection before directory I/O.
func constructQuery(ctx context.Context, config tfsdk.Config) (common.QueryModel, shared.Identity, int, diag.Diagnostics) {
	var model common.QueryModel
	diagnostics := config.GetAttribute(ctx, path.Root("computer_guid"), &model.ComputerGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("result_limit"), &model.ResultLimit)...)
	computer, limit, validation := common.ConstructQuery(model)
	diagnostics.Append(validation...)
	return model, computer, limit, diagnostics
}
