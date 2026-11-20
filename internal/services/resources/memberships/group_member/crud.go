// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// Create adopts a positively observed link or ensures the configured pair through one native permissive addition.
func (r *GroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model GroupMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	group, member, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, group, member)
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError(
			"Membership endpoints unavailable",
			"Both existing endpoint identities must be observed before adoption or addition. No write was submitted.",
		)
		return
	}
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mapState(&model, edge)
	if !edge.Present {
		retained, mutationDiagnostics := r.change(ctx, &model, storage, group, member, id, true)
		resp.Diagnostics.Append(mutationDiagnostics...)
		if !retained {
			return
		}
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
}

// Read retains indeterminate link state and removes state only for positively confirmed exclusion or a missing endpoint GUID.
func (r *GroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model GroupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	group, member, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, group, member)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if edge.Confirmed && !edge.Present {
		resp.State.RemoveResource(ctx)
		return
	}
	if errors.Is(err, common.ErrVisibility) {
		mapState(&model, edge)
		resp.Diagnostics.AddWarning(
			"Membership visibility is indeterminate",
			"An omitted member attribute may be empty or hidden. Identity is retained; an explicit apply will ensure only the configured link using native permissive semantics.",
		)
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
		return
	}
	if err != nil || !edge.Confirmed {
		resp.Diagnostics.AddError(
			"Membership observation incomplete",
			"Known link identity is retained. An omitted attribute or empty search result does not prove absence; peer error text is withheld.",
		)
		return
	}
	mapState(&model, edge)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
}

// Update ensures the same configured pair when refresh could not confirm membership; endpoint changes require replacement.
func (r *GroupMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var previous, plan GroupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	group, member, id, diagnostics := constructPair(previous)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	wantedGroup, wantedMember, validation := common.ConstructPair(plan.GroupGUID, plan.MemberGUID)
	resp.Diagnostics.Append(validation...)
	if wantedGroup.String() != group.String() || wantedMember.String() != member.String() {
		resp.Diagnostics.AddError("Membership replacement required", "A direct link's endpoint GUIDs cannot be updated in place.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	edge, err := common.Read(ctx, r.client, group, member)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError("Membership refresh incomplete", "The existing endpoints could not be confirmed. No addition was submitted.")
	}
	if !resp.Diagnostics.HasError() {
		mapState(&plan, edge)
		if !edge.Confirmed || !edge.Present {
			_, mutationDiagnostics := r.change(ctx, &plan, storage, group, member, id, true)
			resp.Diagnostics.Append(mutationDiagnostics...)
		}
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, plan)...)
	}
}

// Delete removes only the owned direct link after resolving pending effects and confirming current endpoint identities.
func (r *GroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model GroupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	group, member, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, group, member)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if resp.Diagnostics.HasError() || (edge.Confirmed && !edge.Present) {
		return
	}
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError("Membership deletion preflight incomplete", "The owned direct link could not be confirmed. No removal was submitted.")
		return
	}
	_, diagnostics = r.change(ctx, &model, storage, group, member, id, false)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(persistState(ctx, &resp.State, nil, model)...)
	}
}

func (r *GroupMemberResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Membership client unavailable", "Configure the provider before managing direct group links.")
		return false
	}
	return true
}
