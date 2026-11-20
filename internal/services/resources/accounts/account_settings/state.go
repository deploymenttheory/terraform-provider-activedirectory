// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

// mapSnapshot preserves only same-GUID observations, retaining prior known optional fields when the server cannot return them.
func mapSnapshot(ctx context.Context, model *AccountSettingsResourceModel, snapshot common.Snapshot) diag.Diagnostics {
	diagnostics := snapshot.Diagnostics
	if diagnostics.HasError() || snapshot.Values == nil {
		return diagnostics
	}
	id := snapshot.Values["id"].(types.String)
	if !model.ID.IsNull() && !model.ID.IsUnknown() && !model.ID.Equal(id) {
		diagnostics.AddError("Account settings identity changed", "A different account GUID was returned; previous state is retained.")
		return diagnostics
	}
	attributeTypes := common.AttributeTypes()
	attributeTypes["account_guid"] = types.StringType
	previous, conversion := types.ObjectValueFrom(ctx, attributeTypes, model)
	diagnostics.Append(conversion...)
	if diagnostics.HasError() {
		return diagnostics
	}
	values := maps.Clone(snapshot.Values)
	values["account_guid"] = model.AccountGUID
	old := previous.Attributes()
	retained := false
	patch := accounts.AccountControlPatch{}
	for _, field := range flagProperties(model, &patch) {
		if values[field.name].IsNull() && !old[field.name].IsNull() && !old[field.name].IsUnknown() {
			values[field.name] = old[field.name]
			retained = true
		}
	}
	if snapshot.Expiration.Expiration != nil && snapshot.Expiration.Expiration.Value == nil {
		for _, name := range []string{"expires_at", "never_expires"} {
			if !old[name].IsNull() && !old[name].IsUnknown() {
				values[name] = old[name]
				retained = true
			}
		}
	}
	object, conversion := types.ObjectValue(attributeTypes, values)
	diagnostics.Append(conversion...)
	if !diagnostics.HasError() {
		diagnostics.Append(object.As(ctx, model, basetypes.ObjectAsOptions{})...)
	}
	if retained {
		diagnostics.AddWarning(
			"Some account settings were not observable",
			"Prior known optional values are retained without claiming current verification. Hidden values do not authorize a reset or prove a pending write.",
		)
	}
	return diagnostics
}

// persistState stores the adopted identity and known settings before returning any later-phase failure.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model *AccountSettingsResourceModel) diag.Diagnostics {
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() && !model.ID.IsUnknown() {
		diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), model.ID)...)
	}
	return diagnostics
}

// observedFlag converts only a complete visible Boolean, never a null or provider unknown.
func observedFlag(values map[string]attr.Value, name string) (bool, bool) {
	value, ok := values[name].(types.Bool)
	return value.ValueBool(), ok && !value.IsNull() && !value.IsUnknown()
}
