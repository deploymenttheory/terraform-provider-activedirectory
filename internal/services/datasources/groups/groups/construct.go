// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package groups

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/groups"
)

// constructQuery keeps Framework configuration parsing separate from SDK search execution.
func constructQuery(ctx context.Context, config tfsdk.Config) (GroupsDataSourceModel, shared.SearchOptions, diag.Diagnostics) {
	model, diagnostics := common.SearchConfig(ctx, config)
	options, validation := common.ConstructSearch(model)
	diagnostics.Append(validation...)
	return model, options, diagnostics
}
