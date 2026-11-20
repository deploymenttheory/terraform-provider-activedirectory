// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// constructLookup validates both inputs before reading directory objects or evaluating a link assertion.
func constructLookup(ctx context.Context, config tfsdk.Config) (ComputerServiceAccountDataSourceModel, shared.Identity, shared.Identity, diag.Diagnostics) {
	var model ComputerServiceAccountDataSourceModel
	diagnostics := config.GetAttribute(ctx, path.Root("computer_guid"), &model.ComputerGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("service_account_guid"), &model.AccountGUID)...)
	computer, account, validation := common.ConstructPair(model.ComputerGUID, model.AccountGUID)
	diagnostics.Append(validation...)
	return model, computer, account, diagnostics
}
