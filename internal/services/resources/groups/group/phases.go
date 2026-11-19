// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// refresh observes the retained GUID once without retrying any earlier mutation.
func (r *GroupResource) refresh(ctx context.Context, model *GroupResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	target, err := shared.IdentityByGUID(model.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Group GUID unavailable", "Retain recovery_dn and reconcile the earlier creation before another write.")
		return diagnostics
	}
	result, err := r.client.Group(ctx, target, shared.ResolveIdentityOptions{})
	if result.Group != nil {
		diagnostics.Append(mapState(model, result.Group)...)
	}
	if err != nil || !result.Observation.Complete || result.Group == nil {
		diagnostics.AddError("Group readback incomplete", "The earlier operation may already be applied. State and mutation intent are retained; no write is replayed.")
	}
	return diagnostics
}

// recordMutation requires same-GUID raw readback before adopting a new name or location, including verified SDK no-ops.
func recordMutation(model *GroupResourceModel, result shared.ObjectMutationResult, err error) diag.Diagnostics {
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
		diagnostics.AddError("Conflicting group readback identity", "Returned values did not identify the retained group GUID. No values are adopted.")
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

// finishMutation retains intent across failed readback and preserves accepted effects before returning diagnostics.
func (r *GroupResource) finishMutation(ctx context.Context, model *GroupResourceModel, storage pendingData, pending pendingMutation,
	result shared.ObjectMutationResult, err error,
) diag.Diagnostics {
	diagnostics := recordMutation(model, result, err)
	if !diagnostics.HasError() {
		diagnostics.Append(r.refresh(ctx, model)...)
	}
	diagnostics.Append(finishPending(ctx, storage, pending, model, result, err, !diagnostics.HasError())...)
	return diagnostics
}

// applyProperties delegates category/scope read-modify-write and conversion constraints to the SDK, preserving unrelated flags.
func (r *GroupResource) applyProperties(ctx context.Context, current, plan *GroupResourceModel, target shared.Identity, storage pendingData) diag.Diagnostics {
	request, changed, diagnostics := constructUpdate(current, plan, target)
	if diagnostics.HasError() || !changed {
		return diagnostics
	}
	pending := pendingForUpdate(request, plan)
	pending.GUID = current.ID.ValueString()
	diagnostics.Append(savePending(ctx, storage, pending)...)
	if diagnostics.HasError() {
		return diagnostics
	}
	result, err := r.client.UpdateGroup(ctx, request)
	diagnostics.Append(recordMutation(current, result.Mutation, err)...)
	if result.Mutation.State == shared.ObjectWriteAccepted && result.Mutation.TargetGUID != nil && result.Mutation.TargetGUID.String() == current.ID.ValueString() {
		retainAcceptedProperties(current, plan, request)
	}
	if result.Group != nil {
		diagnostics.Append(mapState(current, result.Group)...)
	}
	if !diagnostics.HasError() {
		diagnostics.Append(r.refresh(ctx, current)...)
	}
	diagnostics.Append(finishPending(ctx, storage, pending, current, result.Mutation, err, !diagnostics.HasError())...)
	return diagnostics
}

// retainAcceptedProperties keeps only acknowledged requested fields when later attribute visibility cannot prove them.
func retainAcceptedProperties(current, plan *GroupResourceModel, request groups.UpdateGroupRequest) {
	before, after := propertyBindings(current, &request), propertyBindings(plan, &request)
	for index, field := range before {
		if *field.target != nil {
			*field.value = *after[index].value
		}
		for _, change := range request.Changes {
			if change.Kind == shared.AttributeClear && change.Attribute == field.ldap {
				*field.value = types.StringNull()
			}
		}
	}
	if request.Category != nil {
		current.Category = plan.Category
	}
	if request.Scope != nil {
		current.Scope = plan.Scope
	}
	current.ClearProperties = plan.ClearProperties
}
