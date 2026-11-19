// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Create submits the SDK's disabled/password/enable workflow once and retains accepted stages on later failure.
func (r *ComputerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config ComputerResourceModel
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
	pending := pendingForUpdate(computers.UpdateComputerRequest{Properties: request.Properties, Control: request.Control}, &plan)
	logonIntent := plan.ChangePasswordAtLogon
	if !logonIntent.IsNull() {
		pending.Flags["change_password_at_logon"] = logonIntent.ValueBool()
	}
	pending.Operation, pending.GUID, pending.Name, pending.ParentDN = "create", "", plan.Name.ValueString(), plan.ParentDN.ValueString()
	pending.PasswordVersion = plan.PasswordWOVersion.ValueInt64()
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateComputer(ctx, request)
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
	plan.PasswordWOVersion = types.Int64Value(0)
	if result.Password != nil && result.Password.State == shared.ObjectWriteAccepted && result.Password.TargetGUID != nil &&
		result.Password.TargetGUID.String() == plan.ID.ValueString() && !shared.IsUncertain(err) {
		plan.PasswordWOVersion = types.Int64Value(pending.PasswordVersion)
	}
	if result.Computer != nil && !outcome.ID.IsNull() && !outcome.ID.IsUnknown() {
		resp.Diagnostics.Append(mapState(&plan, result.Computer)...)
	}
	if request.InitialPassword != nil &&
		(result.Password == nil || result.Password.State != shared.ObjectWriteAccepted || plan.PasswordWOVersion.ValueInt64() != pending.PasswordVersion) {
		resp.Diagnostics.AddError(
			"Computer password stage unacknowledged",
			"The SDK did not acknowledge password setup for this GUID. The accepted creation and unresolved intent are retained.",
		)
	}
	if request.Control.Enabled != nil && *request.Control.Enabled && (result.Enable == nil || result.Enable.State != shared.ObjectWriteAccepted || !result.Enable.Verified ||
		result.Enable.TargetGUID == nil || result.Enable.TargetGUID.String() != plan.ID.ValueString()) {
		resp.Diagnostics.AddError("Computer enable stage incomplete", "The SDK did not verify requested enablement. Earlier accepted stages and GUID are retained.")
	}
	if err == nil && !result.Complete {
		resp.Diagnostics.AddError(
			"Computer creation stages incomplete",
			"The SDK did not complete every requested stage. Preserve accepted identity and reconcile before another write.",
		)
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	}
	if !resp.Diagnostics.HasError() && !logonIntent.IsNull() && !plan.ChangePasswordAtLogon.Equal(logonIntent) {
		resp.Diagnostics.Append(r.finishCreationLogonSetting(ctx, &plan, logonIntent, storage, pending)...)
	}
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &plan, result.Creation, err, !resp.Diagnostics.HasError() && result.Complete)...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &plan)...)
}

// Read applies complete or partial observations but removes state only for a pure confirmed absence of the same GUID.
func (r *ComputerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ComputerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Computer GUID needs reconciliation",
			"No stable creation GUID was observed. Preserve recovery_dn and verify the object externally before explicit GUID import; candidates are not adopted.",
		)
		return
	}
	result, err := r.client.Computer(ctx, target, shared.ResolveIdentityOptions{})
	if lifecycle.ConfirmedAbsent(result.Observation, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if result.Computer != nil {
		resp.Diagnostics.Append(mapState(&state, result.Computer)...)
	}
	if err != nil || !result.Observation.Complete || result.Computer == nil {
		resp.Diagnostics.AddError(
			"Computer read incomplete",
			"Known identity and prior values are retained. The SDK did not establish a complete observation; peer error text is withheld.",
		)
	} else {
		var storage pendingData
		if resp.Private != nil {
			storage = resp.Private
		}
		resp.Diagnostics.Append(reconcilePending(ctx, storage, result.Computer)...)
		state.RecoveryDN = types.StringNull()
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &state)...)
}

// Update validates every phase first, then preserves each accepted disable, rename, move, password and property change.
func (r *ComputerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current, plan, config ComputerResourceModel
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
		resp.Diagnostics.AddError("Invalid computer identity", "Updates require the existing nonzero object GUID.")
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
	if plan.PasswordWOVersion.ValueInt64() > current.PasswordWOVersion.ValueInt64() {
		resp.Diagnostics.Append(r.resetPassword(ctx, &current, plan.PasswordWOVersion.ValueInt64(), config.PasswordWO, target, storage)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Password resets change pwdLastSet. Observe that effect before applying explicit logon-password intent.
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(r.applyProperties(ctx, &current, &plan, target, storage)...)
	if !resp.Diagnostics.HasError() {
		current.ClearProperties = plan.ClearProperties
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
	}
}

// Delete submits a leaf-only SDK deletion and retains state whenever absence or same-GUID deletion is unproved.
func (r *ComputerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ComputerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid computer deletion identity", "Deletion requires the existing nonzero GUID.")
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
	result, err := r.client.DeleteComputer(ctx, computers.DeleteComputerRequest{Identity: target, Recursive: false})
	mutation := result.Mutation
	absent := lifecycle.ConfirmedAbsent(shared.IdentityResult{Object: mutation.Before, Candidates: mutation.PreflightCandidates}, err)
	deleted := absent || lifecycle.ConfirmedDeletion(mutation, err)
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &state, mutation, err, deleted)...)
	if deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		"Computer deletion not confirmed",
		"The computer may be protected, nonempty or inaccessible. State is retained; no recursive removal or automatic retry occurs.",
	)
}

func (r *ComputerResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Computer client unavailable", "Configure the Active Directory provider before accessing computers.")
		return false
	}
	return true
}

// refresh performs one read without replaying a write or treating partial observations as success.
func (r *ComputerResource) refresh(ctx context.Context, model *ComputerResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	target, err := shared.IdentityByGUID(model.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Computer GUID unavailable", "Retain recovery_dn and reconcile the earlier creation before another write.")
		return diagnostics
	}
	result, err := r.client.Computer(ctx, target, shared.ResolveIdentityOptions{})
	if result.Computer != nil {
		diagnostics.Append(mapState(model, result.Computer)...)
	}
	if err != nil || !result.Observation.Complete || result.Computer == nil {
		diagnostics.AddError("Computer readback incomplete", "The earlier operation may already be applied. Known state and mutation intent are retained; no write is replayed.")
	}
	return diagnostics
}

// recordMutation requires same-GUID raw readback before adopting a new name or location, including verified SDK no-ops.
func recordMutation(model *ComputerResourceModel, result shared.ObjectMutationResult, err error) diag.Diagnostics {
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
		diagnostics.AddError("Conflicting computer readback identity", "Returned values did not identify the retained computer GUID. No values are adopted.")
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
func (r *ComputerResource) finishMutation(
	ctx context.Context,
	model *ComputerResourceModel,
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
