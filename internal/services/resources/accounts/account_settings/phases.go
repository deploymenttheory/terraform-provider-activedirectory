// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

// finishPhase retains accepted observations and resolves a submitted intent only through a fresh positive same-GUID read.
func (r *AccountSettingsResource) finishPhase(
	ctx context.Context, model *AccountSettingsResourceModel, storage pendingData, pending pendingMutation,
	mutation shared.ObjectMutationResult, writeErr error, fallback common.Snapshot,
) (common.Snapshot, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if returned, conflict := mutationIdentityConflict(mutation, pending.GUID); conflict {
		pending.ReportedGUID = returned
		pending.IdentityConflict = true
		diagnostics.Append(savePending(ctx, storage, pending)...)
		diagnostics.AddError(
			"Conflicting account mutation identity",
			"The write result referred to a different account identity. Original GUID and recovery intent are retained; independently reconcile effects before explicit GUID import.",
		)
		return fallback, diagnostics
	}
	mayHaveEffect := mutation.State == shared.ObjectWriteAccepted || mutation.State == shared.ObjectWriteUncertain || mutation.PartialEffectsPossible ||
		shared.IsUncertain(writeErr)
	if !mayHaveEffect {
		diagnostics.Append(clearPending(ctx, storage)...)
		diagnostics.AddError("Account settings mutation rejected", "The settings phase was not accepted. No later phase was submitted; directory peer text is withheld.")
		return fallback, diagnostics
	}
	if fallback.Values != nil && !fallback.Diagnostics.HasError() {
		diagnostics.Append(mapSnapshot(ctx, model, fallback)...)
	}
	target, _ := shared.IdentityByGUID(pending.GUID)
	refreshed, readErr := common.Read(ctx, r.client, target)
	if readErr == nil {
		diagnostics.Append(mapSnapshot(ctx, model, refreshed)...)
		if !diagnostics.HasError() && pendingMatches(pending, refreshed.Values) {
			diagnostics.Append(clearPending(ctx, storage)...)
			if writeErr != nil || mutation.State != shared.ObjectWriteAccepted || !mutation.Verified {
				diagnostics.AddWarning(
					"Account settings effect independently reconciled",
					"Fresh same-GUID observations match the submitted setting intent. No write was retried or rolled back.",
				)
			}
			return refreshed, diagnostics
		}
	}
	diagnostics.AddError(
		"Account settings write needs reconciliation",
		"A submitted change may have effects, but fresh observations did not prove the same-GUID intent. Known "+
			"identity and effects are retained; no later write or automatic replacement is permitted before reconciliation.",
	)
	return fallback, diagnostics
}
