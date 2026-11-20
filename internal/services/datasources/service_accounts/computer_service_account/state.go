// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// mapState preserves explicit input spelling and emits the common canonical import identity and observed names.
func mapState(ctx context.Context, state *tfsdk.State, model ComputerServiceAccountDataSourceModel, edge common.Edge) diag.Diagnostics {
	values, err := edge.Values()
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Incomplete association identity", "No usable direct-link identity was returned.")}
	}
	values["computer_guid"], values["service_account_guid"] = model.ComputerGUID, model.AccountGUID
	var diagnostics diag.Diagnostics
	for name, value := range values {
		diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diagnostics
}
