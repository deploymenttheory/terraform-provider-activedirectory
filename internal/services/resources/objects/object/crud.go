// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Create creates one explicitly selected schema instance, retaining any accepted GUID or recovery DN after failed readback.
func (r *ObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	changes, properties, diagnostics := constructPlan(&plan, true)
	resp.Diagnostics.Append(diagnostics...)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	acks, diagnostics := loadAcknowledgements(ctx, storage)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	parent, _ := shared.IdentityByDN(plan.ParentDN.ValueString())
	request := shared.CreateObjectRequest{
		Parent: parent, Name: plan.Name.ValueString(), Class: plan.ObjectClass.ValueString(),
		Attributes: map[string][]shared.RawValue{}, Options: shared.ObjectMutationOptions{Properties: properties},
	}
	for _, change := range changes {
		request.Attributes[change.Attribute] = change.Values
	}
	pending := mutationCheckpoint("create", "", changes)
	pending.Name, pending.ParentDN = request.Name, plan.ParentDN.ValueString()
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateObject(ctx, request)
	if result.State != shared.ObjectWriteAccepted && result.State != shared.ObjectWriteUncertain {
		resp.Diagnostics.Append(lifecycle.ObserveMutation(types.StringNull(), result, err).Diagnostics...)
		resp.Diagnostics.Append(finishPending(ctx, storage, pending, result, err, false)...)
		return
	}
	plan.ID, plan.RecoveryDN, plan.NeedsUpdate = types.StringNull(), types.StringNull(), types.BoolValue(false)
	resp.Diagnostics.Append(r.recordMutation(ctx, &plan, nil, storage, pending, result, err, properties, changes, acks)...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &plan)...)
}

// Read keeps configured intent and visibility separate, removing state only for pure confirmed absence.
func (r *ObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Object GUID needs reconciliation", "Preserve recovery_dn and independently verify the existing object before explicit GUID import.")
		return
	}
	properties, diagnostics := selectedProperties(&state)
	resp.Diagnostics.Append(diagnostics...)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	acks, diagnostics := loadAcknowledgements(ctx, storage)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{Properties: properties})
	if lifecycle.ConfirmedAbsent(result, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if result.Object != nil {
		resp.Diagnostics.Append(mapObservation(&state, *result.Object, properties, acks)...)
	}
	if err != nil || !result.Complete || result.Object == nil {
		resp.Diagnostics.AddError("Object read incomplete", "Known identity and raw intent are retained. Directory peer text is withheld; no incomplete result proves absence.")
	} else if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(reconcilePending(ctx, storage, *result.Object, acks)...)
		state.RecoveryDN = types.StringNull()
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &state)...)
}

// Update validates all phases before changing name/location and applying one atomic raw attribute modification.
func (r *ObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current, plan ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	changes, properties, diagnostics := constructPlan(&plan, false)
	resp.Diagnostics.Append(diagnostics...)
	target, err := shared.IdentityByGUID(current.ID.ValueString())
	if err != nil || !strings.EqualFold(current.ObjectClass.ValueString(), plan.ObjectClass.ValueString()) {
		resp.Diagnostics.AddError("Invalid object update identity", "Updates require the existing GUID and instantiated class; changing class requires replacement.")
	}
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	acks, diagnostics := loadAcknowledgements(ctx, storage)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer func() { resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &current)...) }()
	acks = eligibleAcknowledgements(&current, acks)
	observed, diagnostics := r.observe(ctx, current.ID, properties)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(mapObservation(&current, *observed, properties, acks)...)
	effective, full, diagnostics := effectiveChanges(changes, *observed, acks)
	resp.Diagnostics.Append(diagnostics...)
	if !strings.EqualFold(current.ObjectClass.ValueString(), plan.ObjectClass.ValueString()) {
		resp.Diagnostics.AddError("Object class observation changed", "The same GUID did not retain the planned class. No mutation was submitted.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	options := shared.ObjectMutationOptions{Properties: properties}
	if !current.Name.Equal(plan.Name) {
		pending := pendingMutation{Operation: "rename", GUID: current.ID.ValueString(), Name: plan.Name.ValueString()}
		resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
		if resp.Diagnostics.HasError() {
			return
		}
		result, renameErr := r.client.RenameObject(ctx, shared.RenameObjectRequest{Identity: target, Name: plan.Name.ValueString(), Options: options})
		resp.Diagnostics.Append(r.recordMutation(ctx, &current, nil, storage, pending, result, renameErr, properties, nil, acks)...)
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
		result, moveErr := r.client.MoveObject(ctx, shared.MoveObjectRequest{Identity: target, TargetParent: parent, Options: options})
		resp.Diagnostics.Append(r.recordMutation(ctx, &current, nil, storage, pending, result.Mutation, moveErr, properties, nil, acks)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(effective) != 0 {
		pending := mutationCheckpoint("update", current.ID.ValueString(), full)
		resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
		if resp.Diagnostics.HasError() {
			return
		}
		result, updateErr := r.client.UpdateObject(ctx, shared.UpdateObjectRequest{Identity: target, Changes: effective, Options: options})
		resp.Diagnostics.Append(r.recordMutation(ctx, &current, &plan, storage, pending, result, updateErr, properties, full, acks)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	adoptIntent(&current, &plan)
	current.ParentDN, current.ObjectClass, current.NeedsUpdate, current.RecoveryDN = plan.ParentDN, plan.ObjectClass, types.BoolValue(false), types.StringNull()
}

// Delete submits one leaf-only removal and requires verified same-GUID deletion before dropping state.
func (r *ObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid object deletion identity", "Deletion requires the existing nonzero GUID.")
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
	_, diagnostics := loadAcknowledgements(ctx, storage)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.DeleteObject(ctx, shared.DeleteObjectRequest{Identity: target, Recursive: false})
	absent := lifecycle.ConfirmedAbsent(shared.IdentityResult{Object: result.Before, Candidates: result.PreflightCandidates}, err)
	deleted := absent || lifecycle.ConfirmedDeletion(result, err)
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, result, err, deleted)...)
	if deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError("Object deletion not confirmed", "The object may be protected, nonempty or inaccessible. State is retained; no recursive delete or retry occurs.")
}
