// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Create submits the SDK's disabled/password/enable workflow once and retains accepted stages on later failure.
func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config ServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(preparePlan(&plan, &config, nil)...)
	request, diagnostics := constructCreate(&plan, config.PasswordWO)
	defer clear(request.InitialPassword)
	config.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(diagnostics...)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pending := pendingForUpdate(service_accounts.UpdateServiceAccountRequest{Properties: request.Properties, Control: request.Control}, &plan)

	pending.Operation, pending.GUID, pending.Name, pending.ParentDN = "create", "", plan.Name.ValueString(), plan.ParentDN.ValueString()
	pending.PasswordVersion = plan.PasswordWOVersion.ValueInt64()
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateServiceAccount(ctx, request)
	outcome := lifecycle.ObserveMutation(types.StringNull(), result.Creation, err)
	resp.Diagnostics.Append(outcome.Diagnostics...)
	if result.Creation.State != shared.ObjectWriteAccepted && result.Creation.State != shared.ObjectWriteUncertain {
		resp.Diagnostics.Append(finishPending(ctx, storage, pending, &plan, result.Creation, err, false)...)
		return
	}
	plan.ID, plan.RecoveryDN = outcome.ID, types.StringNull()
	if outcome.RecoveryDN != "" {
		plan.RecoveryDN = types.StringValue(outcome.RecoveryDN)
	}
	requestedEnable := request.Control.Enabled == nil || *request.Control.Enabled
	resp.Diagnostics.Append(observeCreationStages(&plan, pending.PasswordVersion, request.Kind, requestedEnable, result, err)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	}

	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &plan, result.Creation, err, !resp.Diagnostics.HasError() && result.Complete)...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &plan)...)
}

// Read applies complete or partial observations but removes state only for a pure confirmed absence of the same GUID.
func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Service account GUID needs reconciliation",
			"No stable creation GUID was observed. Preserve recovery_dn and verify the object externally before explicit GUID import; candidates are not adopted.",
		)
		return
	}
	result, err := r.client.ServiceAccount(ctx, target, shared.ResolveIdentityOptions{})
	if lifecycle.ConfirmedAbsent(result.Observation, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if result.Account != nil {
		resp.Diagnostics.Append(mapState(&state, result.Account)...)
	}
	if err != nil || !result.Observation.Complete || result.Account == nil {
		resp.Diagnostics.AddError(
			"Service account read incomplete",
			"Known identity and prior values are retained. The SDK did not establish a complete observation; peer error text is withheld.",
		)
	} else {
		var storage pendingData
		if resp.Private != nil {
			storage = resp.Private
		}
		resp.Diagnostics.Append(reconcilePending(ctx, storage, result.Account)...)
		state.RecoveryDN = types.StringNull()
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &state)...)
}

// Update validates every phase first, then preserves each accepted disable, rename, move and property change.
func (r *ServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current, plan, config ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(preparePlan(&plan, &config, &current)...)
	resp.Diagnostics.Append(validateUpdate(&current, &plan, config.PasswordWO)...)
	target, err := shared.IdentityByGUID(current.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account identity", "Updates require the existing nonzero object GUID.")
	}
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer func() { resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &current)...) }()
	if !plan.Enabled.IsNull() && !plan.Enabled.ValueBool() && !current.Enabled.Equal(plan.Enabled) {
		disablePlan := current
		disablePlan.Enabled = types.BoolValue(false)
		resp.Diagnostics.Append(r.applyProperties(ctx, &current, &disablePlan, target, storage)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !current.Name.Equal(plan.Name) {
		pending := pendingMutation{Operation: "rename", GUID: current.ID.ValueString(), Name: plan.Name.ValueString()}
		resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
		if resp.Diagnostics.HasError() {
			return
		}
		result, renameErr := r.client.RenameObject(ctx, shared.RenameObjectRequest{Identity: target, Name: plan.Name.ValueString()})
		resp.Diagnostics.Append(r.finishMutation(ctx, &current, storage, pending, result, renameErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !strings.EqualFold(current.ParentDN.ValueString(), plan.ParentDN.ValueString()) {
		parent, _ := shared.IdentityByDN(plan.ParentDN.ValueString())
		pending := pendingMutation{Operation: "move", GUID: current.ID.ValueString(), ParentDN: plan.ParentDN.ValueString()}
		resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
		if resp.Diagnostics.HasError() {
			return
		}
		result, moveErr := r.client.MoveObject(ctx, shared.MoveObjectRequest{Identity: target, TargetParent: parent})
		resp.Diagnostics.Append(r.finishMutation(ctx, &current, storage, pending, result.Mutation, moveErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	current.ParentDN = plan.ParentDN

	owned := ownedPlan(&plan, &config)
	resp.Diagnostics.Append(r.applyProperties(ctx, &current, &owned, target, storage)...)
	if !resp.Diagnostics.HasError() {
		current.ClearProperties = plan.ClearProperties
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
	}
}

// Delete submits a leaf-only SDK deletion and retains state whenever absence or same-GUID deletion is unproved.
func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account deletion identity", "Deletion requires the existing nonzero GUID.")
		return
	}
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pending := pendingMutation{Operation: "delete", GUID: state.ID.ValueString()}
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.DeleteServiceAccount(ctx, service_accounts.DeleteServiceAccountRequest{Identity: target})
	mutation := result.Mutation
	deleted := confirmedDeletion(state.ID.ValueString(), mutation, err)
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &state, mutation, err, deleted)...)
	if deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		"Service account deletion not confirmed",
		"The service account may be protected, nonempty or inaccessible. State is retained; no recursive removal or automatic retry occurs.",
	)
}

// confirmedDeletion binds positive deletion and pure absence proof to the resource's retained GUID.
func confirmedDeletion(guid string, mutation shared.ObjectMutationResult, err error) bool {
	sameGUID := mutation.TargetGUID != nil && mutation.TargetGUID.String() == guid
	absent := (mutation.TargetGUID == nil || sameGUID) && lifecycle.ConfirmedAbsent(shared.IdentityResult{Object: mutation.Before, Candidates: mutation.PreflightCandidates}, err)
	return absent || sameGUID && lifecycle.ConfirmedDeletion(mutation, err)
}

func (r *ServiceAccountResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Service account client unavailable", "Configure the Active Directory provider before accessing service accounts.")
		return false
	}
	return true
}

// refresh performs one read without replaying a write or treating partial observations as success.
func (r *ServiceAccountResource) refresh(ctx context.Context, model *ServiceAccountResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	target, err := shared.IdentityByGUID(model.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Service account GUID unavailable", "Retain recovery_dn and reconcile the earlier creation before another write.")
		return diagnostics
	}
	result, err := r.client.ServiceAccount(ctx, target, shared.ResolveIdentityOptions{})
	if result.Account != nil {
		diagnostics.Append(mapState(model, result.Account)...)
	}
	if err != nil || !result.Observation.Complete || result.Account == nil {
		diagnostics.AddError(
			"Service account readback incomplete",
			"The earlier operation may already be applied. Known state and mutation intent are retained; no write is replayed.",
		)
	}
	return diagnostics
}

// recordMutation requires same-GUID raw readback before adopting a new name or location, including verified SDK no-ops.
func recordMutation(model *ServiceAccountResourceModel, result shared.ObjectMutationResult, err error) diag.Diagnostics {
	verifiedNoop := err == nil && result.State == shared.ObjectWriteNotAttempted && result.Verified && result.ReadbackComplete &&
		result.TargetGUID != nil && result.TargetGUID.String() == model.ID.ValueString() && result.After != nil
	var diagnostics diag.Diagnostics
	if !verifiedNoop {
		outcome := lifecycle.ObserveMutation(model.ID, result, err)
		model.ID = outcome.ID
		diagnostics.Append(outcome.Diagnostics...)
	}
	if result.After == nil {
		return diagnostics
	}
	value, valueErr := result.After.Values("objectGUID").Single()
	guid, guidErr := value.GUID()
	if valueErr != nil || guidErr != nil || guid.String() != model.ID.ValueString() {
		diagnostics.AddError("Conflicting service account readback identity", "Returned values did not identify the retained service account GUID. No values are adopted.")
		return diagnostics
	}
	if parent, valid := identity.ParentDN(result.After.DistinguishedName); valid {
		model.ParentDN, model.DistinguishedName = types.StringValue(parent), types.StringValue(result.After.DistinguishedName)
	}
	if name := lifecycle.ObserveString(*result.After, "name"); name.Usable {
		model.Name = name.Value
	}
	return diagnostics
}

// finishMutation retains pending intent through failed readback, including failures after SDK acknowledgement.
func (r *ServiceAccountResource) finishMutation(
	ctx context.Context,
	model *ServiceAccountResourceModel,
	storage pendingData,
	pending pendingMutation,
	result shared.ObjectMutationResult,
	err error,
) diag.Diagnostics {
	diagnostics := recordMutation(model, result, err)
	if !diagnostics.HasError() {
		diagnostics.Append(r.refresh(ctx, model)...)
	}
	diagnostics.Append(finishPending(ctx, storage, pending, model, result, err, !diagnostics.HasError())...)
	return diagnostics
}
