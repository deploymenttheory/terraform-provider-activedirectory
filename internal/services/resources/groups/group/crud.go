// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Create submits one typed group creation and retains accepted identity when subsequent observation fails.
func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(preparePlan(&plan, &config)...)
	request, diagnostics := constructCreate(&plan)
	resp.Diagnostics.Append(diagnostics...)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(guardPending(ctx, storage)...)
	if resp.Diagnostics.HasError() {
		return
	}
	category, scope := plan.Category.ValueString(), plan.Scope.ValueString()
	pending := pendingForUpdate(groups.UpdateGroupRequest{
		SAMAccountName: request.SAMAccountName, Description: request.Description, DisplayName: request.DisplayName,
		HomePage: request.HomePage, ManagedByDN: request.ManagedByDN,
	}, &plan)
	pending.Operation, pending.GUID, pending.Name, pending.ParentDN = "create", "", request.Name, plan.ParentDN.ValueString()
	pending.Category, pending.Scope = &category, &scope
	resp.Diagnostics.Append(savePending(ctx, storage, pending)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateGroup(ctx, request)
	mutation := result.Mutation
	outcome := lifecycle.ObserveMutation(types.StringNull(), mutation, err)
	resp.Diagnostics.Append(outcome.Diagnostics...)
	if mutation.State != shared.ObjectWriteAccepted && mutation.State != shared.ObjectWriteUncertain {
		resp.Diagnostics.Append(finishPending(ctx, storage, pending, &plan, mutation, err, false)...)
		return
	}
	plan.ID, plan.RecoveryDN = outcome.ID, types.StringNull()
	if outcome.RecoveryDN != "" {
		plan.RecoveryDN = types.StringValue(outcome.RecoveryDN)
	}
	if result.Group != nil && !plan.ID.IsNull() {
		resp.Diagnostics.Append(mapState(&plan, result.Group)...)
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	}
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &plan, mutation, err, !resp.Diagnostics.HasError())...)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &plan)...)
}

// Read removes state only after pure confirmed absence, preserving known identity and partial observations on other failures.
func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Group GUID needs reconciliation",
			"Preserve recovery_dn and verify the object before explicit GUID import. DN-only candidates are never adopted.",
		)
		return
	}
	result, err := r.client.Group(ctx, target, shared.ResolveIdentityOptions{})
	if lifecycle.ConfirmedAbsent(result.Observation, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if result.Group != nil {
		resp.Diagnostics.Append(mapState(&state, result.Group)...)
	}
	if err != nil || !result.Observation.Complete || result.Group == nil {
		resp.Diagnostics.AddError("Group read incomplete", "Known identity and values are retained. A complete observation was not established; peer error text is withheld.")
	} else {
		var storage pendingData
		if resp.Private != nil {
			storage = resp.Private
		}
		resp.Diagnostics.Append(reconcilePending(ctx, storage, result.Group)...)
		state.RecoveryDN = types.StringNull()
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &state)...)
}

// Update validates all phases before renaming, moving and applying typed property/category/scope changes to the same GUID.
func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current, plan, config GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(preparePlan(&plan, &config)...)
	resp.Diagnostics.Append(validatePlan(&plan)...)
	target, err := shared.IdentityByGUID(current.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group update identity", "Updates require the existing nonzero group GUID.")
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
	resp.Diagnostics.Append(r.applyProperties(ctx, &current, &plan, target, storage)...)
	if !resp.Diagnostics.HasError() {
		current.ClearProperties = plan.ClearProperties
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
	}
}

// Delete performs one leaf-only group deletion. Group members are separate objects and are never deleted by this resource.
func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid group deletion identity", "Deletion requires the existing nonzero group GUID.")
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
	result, err := r.client.DeleteGroup(ctx, groups.DeleteGroupRequest{Identity: target})
	mutation := result.Mutation
	absent := lifecycle.ConfirmedAbsent(shared.IdentityResult{Object: mutation.Before, Candidates: mutation.PreflightCandidates}, err)
	deleted := absent || lifecycle.ConfirmedDeletion(mutation, err)
	resp.Diagnostics.Append(finishPending(ctx, storage, pending, &state, mutation, err, deleted)...)
	if deleted {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		"Group deletion not confirmed",
		"The group may be protected, nonempty or inaccessible. State is retained; no recursive removal or automatic retry occurs.",
	)
}

func (r *GroupResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Group client unavailable", "Configure the Active Directory provider before accessing groups.")
		return false
	}
	return true
}
