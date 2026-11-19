// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// lookupModel reads only input attributes; computed observation fields are projected independently.
func lookupModel(ctx context.Context, config tfsdk.Config) (ObjectDataSourceModel, diag.Diagnostics) {
	var model ObjectDataSourceModel
	diagnostics := config.GetAttribute(ctx, path.Root("object_guid"), &model.ObjectGUID)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("dn"), &model.DN)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("properties"), &model.Properties)...)
	return model, diagnostics
}
