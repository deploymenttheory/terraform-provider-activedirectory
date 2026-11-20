// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// Create adopts a positively observed link or ensures the configured pair through one native permissive addition.
func (r *ComputerServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model ComputerServiceAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	computer, account, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, computer, account)
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError(
			"Association endpoints unavailable",
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
		retained, mutationDiagnostics := r.change(ctx, &model, storage, computer, account, id, true)
		resp.Diagnostics.Append(mutationDiagnostics...)
		if !retained {
			return
		}
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
}

// Read retains indeterminate link state and removes state only for positively confirmed exclusion or a missing endpoint GUID.
func (r *ComputerServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model ComputerServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	computer, account, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, computer, account)
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
			"Association visibility is indeterminate",
			"An omitted account attribute may be empty or hidden. Identity is retained; an explicit apply will ensure only the configured link using native permissive semantics.",
		)
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
		return
	}
	if err != nil || !edge.Confirmed {
		resp.Diagnostics.AddError(
			"Association observation incomplete",
			"Known link identity is retained. An omitted attribute or empty search result does not prove absence; peer error text is withheld.",
		)
		return
	}
	mapState(&model, edge)
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, model)...)
}

// Update ensures the same configured pair when refresh could not confirm association; endpoint changes require replacement.
func (r *ComputerServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var previous, plan ComputerServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	computer, account, id, diagnostics := constructPair(previous)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	wantedComputer, wantedAccount, validation := common.ConstructPair(plan.ComputerGUID, plan.AccountGUID)
	resp.Diagnostics.Append(validation...)
	if wantedComputer.String() != computer.String() || wantedAccount.String() != account.String() {
		resp.Diagnostics.AddError("Association replacement required", "A direct link's endpoint GUIDs cannot be updated in place.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	edge, err := common.Read(ctx, r.client, computer, account)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError("Association refresh incomplete", "The existing endpoints could not be confirmed. No addition was submitted.")
	}
	if !resp.Diagnostics.HasError() {
		mapState(&plan, edge)
		if !edge.Confirmed || !edge.Present {
			_, mutationDiagnostics := r.change(ctx, &plan, storage, computer, account, id, true)
			resp.Diagnostics.Append(mutationDiagnostics...)
		}
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, plan)...)
	}
}

// Delete removes only the owned direct link after resolving pending effects and confirming current endpoint identities.
func (r *ComputerServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model ComputerServiceAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	computer, account, id, diagnostics := constructPair(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	edge, err := common.Read(ctx, r.client, computer, account)
	var storage pendingData
	if resp.Private != nil {
		storage = resp.Private
	}
	resp.Diagnostics.Append(reconcilePending(ctx, storage, id, edge)...)
	if resp.Diagnostics.HasError() || (edge.Confirmed && !edge.Present) {
		return
	}
	if err != nil && !errors.Is(err, common.ErrVisibility) {
		resp.Diagnostics.AddError("Association deletion preflight incomplete", "The owned direct link could not be confirmed. No removal was submitted.")
		return
	}
	_, diagnostics = r.change(ctx, &model, storage, computer, account, id, false)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(persistState(ctx, &resp.State, nil, model)...)
	}
}

func (r *ComputerServiceAccountResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Association client unavailable", "Configure the provider before managing direct computer links.")
		return false
	}
	return true
}
