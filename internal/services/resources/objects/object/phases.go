// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// observe performs one same-GUID complete read without retrying a preceding mutation or adopting raw candidates.
func (r *ObjectResource) observe(ctx context.Context, id types.String, properties []string) (*shared.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	target, err := shared.IdentityByGUID(id.ValueString())
	if err != nil || ctx.Err() != nil {
		diagnostics.AddError("Object observation unavailable", "A known GUID and active request context are required. Preserve recovery identity before another write.")
		return nil, diagnostics
	}
	result, err := r.client.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{Properties: properties})
	if result.Object != nil {
		if returned, valid := common.GUID(*result.Object); !valid || returned != id.ValueString() {
			diagnostics.AddError("Conflicting object observation", "Returned values do not identify the retained GUID. No observation is adopted.")
			return nil, diagnostics
		}
	}
	if err != nil || !result.Complete || result.Object == nil {
		diagnostics.AddError("Object observation incomplete", "A complete same-GUID observation was not established. Known state is retained; peer text is withheld.")
		return nil, diagnostics
	}
	return result.Object, diagnostics
}

// recordMutation preserves accepted identity and raw observations, retaining checkpoints across any incomplete phase.
func (r *ObjectResource) recordMutation(ctx context.Context, model, plan *ObjectResourceModel, storage pendingData, pending pendingMutation,
	result shared.ObjectMutationResult, err error, properties []string, full []shared.AttributeChange, acks acknowledgements,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	previous := model.ID
	verifiedNoop := err == nil && result.State == shared.ObjectWriteNotAttempted && result.Verified && result.ReadbackComplete && result.After != nil &&
		result.TargetGUID != nil && result.TargetGUID.String() == previous.ValueString()
	if !verifiedNoop {
		outcome := lifecycle.ObserveMutation(previous, result, err)
		model.ID = outcome.ID
		if outcome.RecoveryDN != "" {
			model.RecoveryDN = types.StringValue(outcome.RecoveryDN)
		}
		diagnostics.Append(outcome.Diagnostics...)
	}
	pending.IdentityConflict = mutationIdentityConflict(previous, model.ID, result)
	if plan != nil && !pending.IdentityConflict && (result.State == shared.ObjectWriteAccepted || result.State == shared.ObjectWriteUncertain || verifiedNoop) {
		adoptIntent(model, plan)
	}
	if result.After != nil && !model.ID.IsNull() {
		diagnostics.Append(mapObservation(model, *result.After, properties, acks)...)
	}
	diagnostics.Append(mutationReadbackDiagnostics(result, full, pending.IdentityConflict)...)
	if !diagnostics.HasError() {
		pending.Acknowledged = true
		for _, change := range full {
			hashed := hashedChange(change)
			acks[changeKey(hashed)] = acknowledgedObservation{
				GUID: model.ID.ValueString(), IntentHash: intentHash(hashed),
				ObservationHash: observationHash(*result.After, change.Attribute),
			}
		}
		if len(full) != 0 {
			diagnostics.Append(saveAcknowledgements(ctx, storage, acks)...)
		}
		diagnostics.Append(mapObservation(model, *result.After, properties, acks)...)
	}
	pending.GUID = model.ID.ValueString()
	diagnostics.Append(finishPending(ctx, storage, pending, result, err, !diagnostics.HasError())...)
	return diagnostics
}

// mutationCheckpoint records hashed declared intent rather than only a filtered subset of add values.
func mutationCheckpoint(operation, guid string, changes []shared.AttributeChange) pendingMutation {
	pending := pendingMutation{Operation: operation, GUID: guid}
	for _, change := range changes {
		pending.Changes = append(pending.Changes, hashedChange(change))
	}
	return pending
}

// configured prevents every generic operation from falling back to an unqualified or newly opened session.
func (r *ObjectResource) configured(diagnostics *diag.Diagnostics) bool {
	if r.client == nil {
		diagnostics.AddError("Object client unavailable", "Configure the Active Directory provider before accessing generic objects.")
		return false
	}
	return true
}

// mutationIdentityConflict prevents recovery from promoting another GUID's readback or acknowledged target.
func mutationIdentityConflict(previous, current types.String, result shared.ObjectMutationResult) bool {
	if result.TargetGUID != nil && !previous.IsNull() && !previous.IsUnknown() && result.TargetGUID.String() != previous.ValueString() {
		return true
	}
	if result.After != nil {
		if guid, valid := common.GUID(*result.After); valid && !current.IsNull() && guid != current.ValueString() {
			return true
		}
	}
	return false
}

// mutationReadbackDiagnostics requires a complete same-GUID observation before accepted snapshots can become acknowledgements.
func mutationReadbackDiagnostics(result shared.ObjectMutationResult, full []shared.AttributeChange, conflict bool) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if !result.ReadbackComplete || result.After == nil || conflict {
		diagnostics.AddError("Object mutation readback incomplete", "Identity and accepted effects are retained. Reconcile the earlier submission before another write.")
	}
	if result.After != nil {
		for _, change := range full {
			state := result.After.Values(change.Attribute).State
			if state == shared.AttributePartial || state == shared.AttributeUnloaded {
				diagnostics.AddError("Object mutation attribute readback incomplete", "A changed raw attribute was ranged or unloaded. Pending intent is retained.")
			}
		}
	}
	return diagnostics
}
