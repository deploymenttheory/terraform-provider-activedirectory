// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

// constructQuery validates inputs and lowers SDK materialization to Terraform's requested result bound.
func constructQuery(ctx context.Context, req list.ListRequest) (shared.SearchOptions, diag.Diagnostics) {
	model, diagnostics := common.SearchConfig(ctx, req.Config)
	options, validation := common.ConstructSearch(model)
	diagnostics.Append(validation...)
	if req.Limit < 0 {
		diagnostics.AddError("Invalid discovery limit", "Terraform's result limit cannot be negative.")
	}
	if req.Limit > 0 && req.Limit < int64(options.ResultLimit) {
		options.ResultLimit = int(req.Limit)
	}
	return options, diagnostics
}
