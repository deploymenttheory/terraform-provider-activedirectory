// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// constructQuery validates inputs and lowers SDK materialization to Terraform's requested result bound.
func constructQuery(ctx context.Context, req list.ListRequest) (shared.SearchOptions, diag.Diagnostics) {
	var model ObjectListModel
	diagnostics := req.Config.Get(ctx, &model)
	properties := types.SetNull(types.StringType)
	if model.Properties.IsUnknown() {
		properties = types.SetUnknown(types.StringType)
	} else if !model.Properties.IsNull() {
		properties = types.SetValueMust(types.StringType, model.Properties.Elements())
	}
	options, validation := common.ConstructSearch(common.SearchModel{
		BaseDN: model.BaseDN, LDAPFilter: model.LDAPFilter, Scope: model.Scope,
		PageSize: model.PageSize, ResultLimit: model.ResultLimit, Properties: properties,
	})
	diagnostics.Append(validation...)
	if req.Limit < 0 {
		diagnostics.AddError("Invalid discovery limit", "Terraform's result limit cannot be negative.")
	}
	if req.Limit > 0 && req.Limit < int64(options.ResultLimit) {
		options.ResultLimit = int(req.Limit)
	}
	return options, diagnostics
}
