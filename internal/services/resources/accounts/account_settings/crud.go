// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

// Create adopts an existing account, records its GUID, and applies only configured settings; it never creates a directory object.
func (r *AccountSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model, config AccountSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	intent, diagnostics := constructSettings(&config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ID = types.StringUnknown()
	model.AccountGUID = config.AccountGUID
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	adopted, diagnostics := r.applySettings(ctx, &model, intent, storage)
	resp.Diagnostics.Append(diagnostics...)
	if adopted {
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &model)...)
	}
}

// Read observes the same GUID without owning new fields; only pure confirmed absence removes management state.
func (r *AccountSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model AccountSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	guid, err := identity.CanonicalGUID(model.ID.ValueString())
	configured, configuredErr := identity.CanonicalGUID(model.AccountGUID.ValueString())
	if err != nil || configuredErr != nil || guid != configured {
		resp.Diagnostics.AddError("Invalid adopted account identity", "The retained ID and account_guid must identify the same existing account.")
		return
	}
	target, _ := shared.IdentityByGUID(guid)
	snapshot, err := common.Read(ctx, r.client, target)
	if snapshot.ConfirmedAbsent {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.Append(snapshot.Diagnostics...)
		resp.Diagnostics.AddError(
			"Account settings observation failed",
			"Existing state is retained. Confirm visibility and connectivity to the selected account; directory peer text is withheld.",
		)
		return
	}
	resp.Diagnostics.Append(mapSnapshot(ctx, &model, snapshot)...)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, guid, snapshot.Values)...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &model)...)
}

// Update changes only configured settings after complete preflight and reconciliation of any prior submission.
func (r *AccountSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model, config AccountSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	intent, diagnostics := constructSettings(&config)
	resp.Diagnostics.Append(diagnostics...)
	if intent.GUID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Adopted account target changed", "Target changes require replacement of management, never mutation of another account through this state.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	model.AccountGUID = config.AccountGUID
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	_, diagnostics = r.applySettings(ctx, &model, intent, storage)
	resp.Diagnostics.Append(diagnostics...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &model)...)
}

// Delete relinquishes resolved ownership without a directory deletion or reset; unresolved writes cannot escape through automatic replacement.
func (*AccountSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	pending, diagnostics := loadPending(ctx, storage)
	resp.Diagnostics.Append(diagnostics...)
	if pending != nil {
		resp.Diagnostics.AddError(
			"Account settings recovery required before relinquishment",
			"A prior write is unresolved. Preserve this state and independently reconcile the account before explicit GUID "+
				"import. Automatic replacement cannot discard the checkpoint and replay a change.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.RemoveResource(ctx)
}

// configured refuses directory operations when the provider did not bind the required SDK service.
func (r *AccountSettingsResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Account settings client unavailable", "Configure a qualified Active Directory provider before reading or updating account settings.")
		return false
	}
	return true
}
