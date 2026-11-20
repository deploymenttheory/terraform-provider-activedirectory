// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// constructQuery validates the whole request and lowers the materialization bound to Terraform's requested limit.
func constructQuery(ctx context.Context, req list.ListRequest) (shared.Identity, int, diag.Diagnostics) {
	var model common.QueryModel
	diagnostics := req.Config.Get(ctx, &model)
	computer, limit, validation := common.ConstructQuery(model)
	diagnostics.Append(validation...)
	if req.Limit < 0 {
		diagnostics.AddError("Invalid association query limit", "Terraform's query result limit cannot be negative.")
	}
	if req.Limit > 0 && req.Limit < int64(limit) {
		limit = int(req.Limit)
	}
	return computer, limit, diagnostics
}
