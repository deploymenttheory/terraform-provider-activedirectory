// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

// Read emits typed settings only after both SDK observations resolve to the same existing account GUID.
func (d *AccountSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var guid, dn types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("object_guid"), &guid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("dn"), &dn)...)
	target, diagnostics := common.ConstructLookup(guid, dn)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Account settings client unavailable", "Configure the provider before reading account settings.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	snapshot, err := common.Read(ctx, d.client, target)
	resp.Diagnostics.Append(snapshot.Diagnostics...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Account settings observation failed",
			"The selected account could not be completely resolved. No settings were changed; directory peer text is withheld.",
		)
		return
	}
	for name, value := range snapshot.Values {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_guid"), guid)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("dn"), dn)...)
}
