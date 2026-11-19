// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Create submits one OU creation and retains identity even when protection or readback fails afterward.
func (r *OrganizationalUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationalUnitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	request, diagnostics := constructCreate(&plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateOrganizationalUnit(ctx, request)
	outcome := lifecycle.ObserveMutation(types.StringNull(), result.Object, err)
	if result.Object.State != shared.ObjectWriteAccepted && result.Object.State != shared.ObjectWriteUncertain {
		resp.Diagnostics.Append(outcome.Diagnostics...)
		return
	}
	plan.ID = outcome.ID
	plan.RecoveryDN = types.StringNull()
	if outcome.RecoveryDN != "" {
		plan.RecoveryDN = types.StringValue(outcome.RecoveryDN)
	}
	plan.ProtectedFromAccidentalDeletion = types.BoolNull()
	if result.Protection.ProtectedFromAccidentalDeletion != nil {
		plan.ProtectedFromAccidentalDeletion = types.BoolValue(*result.Protection.ProtectedFromAccidentalDeletion)
	}
	if result.Unit != nil {
		resp.Diagnostics.Append(mapState(&plan, result.Unit)...)
	}
	resp.Diagnostics.Append(outcome.Diagnostics...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &plan)...)
}

// Read refreshes by GUID and removes state only for a pure confirmed absence of that stable identity.
func (r *OrganizationalUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationalUnitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	var target shared.Identity
	var err error
	knownGUID := !state.ID.IsNull() && !state.ID.IsUnknown()
	switch {
	case knownGUID:
		target, err = shared.IdentityByGUID(state.ID.ValueString())
	case !state.RecoveryDN.IsNull() && !state.RecoveryDN.IsUnknown():
		target, err = shared.IdentityByDN(state.RecoveryDN.ValueString())
	default:
		err = shared.ErrInvalidIdentity
	}
	if err != nil {
		resp.Diagnostics.AddError("Missing OU identity", "A valid object GUID or retained creation recovery DN is required. Reconcile state before another write.")
		return
	}
	result, err := r.client.OrganizationalUnit(ctx, target, shared.ResolveIdentityOptions{Properties: []string{"ProtectedFromAccidentalDeletion"}})
	if knownGUID && lifecycle.ConfirmedAbsent(result.Observation, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if result.Unit != nil {
		resp.Diagnostics.Append(mapState(&state, result.Unit)...)
	}
	if err != nil || !result.Observation.Complete || result.Unit == nil {
		resp.Diagnostics.AddError(
			"OU read incomplete",
			"The directory did not establish a complete OU observation. Known identity and prior state are retained; peer error text is withheld.",
		)
	} else {
		state.RecoveryDN = types.StringNull()
	}
	resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &state)...)
}

// Update preserves the GUID through explicit unprotection, rename, same-domain move and property phases.
// Each accepted phase is retained before a later phase can fail; no phase is automatically replayed or rolled back.
func (r *OrganizationalUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current, plan OrganizationalUnitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(validatePlan(&plan)...)
	target, err := shared.IdentityByGUID(current.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid OU identity", "Updates require the existing object GUID; reconcile pending creation before another write.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	defer func() { resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &current)...) }()
	if !plan.ProtectedFromAccidentalDeletion.ValueBool() && !current.ProtectedFromAccidentalDeletion.Equal(plan.ProtectedFromAccidentalDeletion) {
		protect := false
		result, updateErr := r.client.UpdateOrganizationalUnit(ctx, organizational_units.UpdateOrganizationalUnitRequest{
			Identity: target, ProtectedFromAccidentalDeletion: &protect,
		})
		resp.Diagnostics.Append(recordOUResult(&current, result, updateErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !current.Name.Equal(plan.Name) {
		result, renameErr := r.client.RenameObject(ctx, shared.RenameObjectRequest{Identity: target, Name: plan.Name.ValueString()})
		resp.Diagnostics.Append(recordMutation(&current, result, renameErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !strings.EqualFold(current.ParentDN.ValueString(), plan.ParentDN.ValueString()) {
		parent, _ := shared.IdentityByDN(plan.ParentDN.ValueString())
		result, moveErr := r.client.MoveObject(ctx, shared.MoveObjectRequest{Identity: target, TargetParent: parent})
		resp.Diagnostics.Append(recordMutation(&current, result.Mutation, moveErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	current.ParentDN = plan.ParentDN
	if hasPropertyChanges(&current, &plan) {
		request := constructUpdate(&current, &plan, target)
		result, updateErr := r.client.UpdateOrganizationalUnit(ctx, request)
		if result.Object.State == shared.ObjectWriteAccepted {
			copyPropertyIntent(&current, &plan)
		}
		resp.Diagnostics.Append(recordOUResult(&current, result, updateErr)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(r.refresh(ctx, &current)...)
	}
}

// Delete requests leaf deletion only and leaves protection changes to an explicit prior configuration apply.
func (r *OrganizationalUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationalUnitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.configured(&resp.Diagnostics) {
		return
	}
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid OU identity", "Deletion requires the existing object GUID; reconcile pending creation before another mutation.")
		return
	}
	result, err := r.client.DeleteOrganizationalUnit(ctx, organizational_units.DeleteOrganizationalUnitRequest{Identity: target, Recursive: false})
	absent := lifecycle.ConfirmedAbsent(shared.IdentityResult{Object: result.Before, Candidates: result.PreflightCandidates}, err)
	if absent || lifecycle.ConfirmedDeletion(result, err) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		"OU deletion not confirmed",
		"The OU may be protected, nonempty, inaccessible or partly observed. State is retained. Resolve protection and child ownership explicitly before another leaf delete.",
	)
}

func (r *OrganizationalUnitResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("OU client unavailable", "Configure the Active Directory provider before accessing organizational units.")
		return false
	}
	return true
}

// refresh performs one read after an accepted phase, preserving partial observations on failure.
func (r *OrganizationalUnitResource) refresh(ctx context.Context, state *OrganizationalUnitResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	target, err := shared.IdentityByGUID(state.ID.ValueString())
	if err != nil {
		diagnostics.AddError("OU GUID unavailable", "Reconcile the retained creation recovery DN before another write.")
		return diagnostics
	}
	result, err := r.client.OrganizationalUnit(ctx, target, shared.ResolveIdentityOptions{Properties: []string{"ProtectedFromAccidentalDeletion"}})
	if result.Unit != nil {
		diagnostics.Append(mapState(state, result.Unit)...)
	}
	if err != nil || !result.Observation.Complete || result.Unit == nil {
		diagnostics.AddError("OU readback incomplete", "The earlier directory operation may already be applied. Retain identity and reconcile before another write.")
	}
	return diagnostics
}

// recordMutation retains accepted generic rename/move results without borrowing another object's identity.
func recordMutation(state *OrganizationalUnitResourceModel, result shared.ObjectMutationResult, err error) diag.Diagnostics {
	outcome := lifecycle.ObserveMutation(state.ID, result, err)
	state.ID = outcome.ID
	if result.After != nil && result.After.DistinguishedName != "" {
		state.DistinguishedName = types.StringValue(result.After.DistinguishedName)
	}
	return outcome.Diagnostics
}

// recordOUResult keeps ACL-only outcomes distinct from the generic attribute Modify outcome.
func recordOUResult(state *OrganizationalUnitResourceModel, result organizational_units.OrganizationalUnitMutationResult, err error) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if result.Object.State != shared.ObjectWriteNotAttempted {
		diagnostics.Append(recordMutation(state, result.Object, err)...)
	} else if err != nil || result.Protection.ProtectedFromAccidentalDeletion == nil {
		diagnostics.AddError("OU protection change incomplete", "Retain the object GUID and reconcile the individual SDK protection outcomes before another write.")
	}
	if result.Protection.ProtectedFromAccidentalDeletion != nil {
		state.ProtectedFromAccidentalDeletion = types.BoolValue(*result.Protection.ProtectedFromAccidentalDeletion)
	}
	if err != nil && result.Unit != nil {
		diagnostics.Append(mapState(state, result.Unit)...)
	}
	return diagnostics
}

func copyPropertyIntent(current, plan *OrganizationalUnitResourceModel) {
	ignored := organizational_units.OUProperties{}
	before, after := propertyBindings(current, &ignored), propertyBindings(plan, &ignored)
	for index, field := range after {
		*before[index].value = *field.value
	}
}
