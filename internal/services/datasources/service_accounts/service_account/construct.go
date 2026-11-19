// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// lookupModel reads only input attributes; computed observation fields are projected independently.
func lookupModel(ctx context.Context, config tfsdk.Config) (ServiceAccountDataSourceModel, diag.Diagnostics) {
	var model ServiceAccountDataSourceModel
	diagnostics := config.GetAttribute(ctx, path.Root("object_guid"), &model.ObjectGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("dn"), &model.DN)...)
	return model, diagnostics
}
