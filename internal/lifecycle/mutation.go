// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package lifecycle

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// MutationOutcome separates acknowledged effects, retained identity and required reconciliation.
// RecoveryDN is a locator for an accepted creation whose GUID could not be observed; it is never a GUID substitute.
type MutationOutcome struct {
	ID                types.String
	RecoveryDN        string
	Accepted          bool
	ReconcileRequired bool
	Diagnostics       diag.Diagnostics
}

// ObserveMutation preserves stable identity even when SDK readback or later workflow steps fail.
// It performs no I/O and never authorizes mutation replay or automatic rollback.
func ObserveMutation(previous types.String, result shared.ObjectMutationResult, err error) MutationOutcome {
	outcome := MutationOutcome{
		ID: previous, Accepted: result.State == shared.ObjectWriteAccepted,
		ReconcileRequired: result.State == shared.ObjectWriteUncertain || result.PartialEffectsPossible,
	}
	if outcome.ID.IsUnknown() {
		outcome.ID = types.StringNull()
	}
	if !outcome.observeIdentity(result) {
		return outcome
	}
	if (outcome.Accepted || result.State == shared.ObjectWriteUncertain) && outcome.ID.IsNull() {
		outcome.RecoveryDN = result.TargetDN
		outcome.ReconcileRequired = true
		outcome.Diagnostics.AddError("Directory creation needs reconciliation",
			"The directory may contain the created object, but its GUID was not observed. Preserve the recovery DN and resolve the existing object before another create.")
	}
	if err == nil && outcome.Accepted && result.Verified && !outcome.ReconcileRequired {
		return outcome
	}
	if outcome.Accepted && shared.Classify(err) == shared.ClassUnverified && !outcome.ReconcileRequired {
		outcome.Diagnostics.AddWarning("Directory write accepted without readback proof",
			"The server accepted the change. Attribute visibility does not prove its effect; retain accepted intent without claiming independent verification.")
		return outcome
	}
	if err != nil || !outcome.Accepted || !result.Verified {
		outcome.ReconcileRequired = outcome.ReconcileRequired || outcome.Accepted || shared.IsUncertain(err)
		outcome.Diagnostics.AddError("Directory mutation did not complete",
			"Retain known identity and observations. Reconcile accepted or uncertain effects before another write. Directory peer error text is withheld.")
	}
	return outcome
}

// observeIdentity reconciles authoritative target/readback GUIDs while preserving any existing identity.
func (outcome *MutationOutcome) observeIdentity(result shared.ObjectMutationResult) bool {
	if !outcome.ID.IsNull() {
		canonical, parseErr := identity.CanonicalGUID(outcome.ID.ValueString())
		if parseErr != nil {
			outcome.Diagnostics.AddError("Invalid stored identity", "The existing object GUID is invalid. Reconcile state before another mutation.")
			outcome.ReconcileRequired = true
			return false
		}
		outcome.ID = types.StringValue(canonical)
	}
	target := result.TargetGUID
	if result.After != nil {
		if value, valueErr := result.After.Values("objectGUID").Single(); valueErr == nil {
			if guid, guidErr := value.GUID(); guidErr == nil && guid != (shared.GUID{}) {
				if target != nil && *target != guid {
					if outcome.ID.IsNull() && *target != (shared.GUID{}) && (outcome.Accepted || result.State == shared.ObjectWriteUncertain) {
						outcome.ID = types.StringValue(target.String())
					}
					outcome.ReconcileRequired = true
					outcome.Diagnostics.AddError("Conflicting directory identities",
						"The mutation target and readback have different GUIDs. Preserve existing identity and reconcile before another write.")
					return false
				}
				target = &guid
			}
		}
	}
	if target != nil && *target != (shared.GUID{}) &&
		(outcome.Accepted || result.State == shared.ObjectWriteUncertain || !outcome.ID.IsNull()) {
		returned := target.String()
		if !outcome.ID.IsNull() && outcome.ID.ValueString() != returned {
			outcome.Diagnostics.AddError("Directory identity changed",
				"The mutation returned a different object GUID. Existing identity is retained; reconcile before another write.")
			outcome.ReconcileRequired = true
			return false
		}
		outcome.ID = types.StringValue(returned)
	}
	return true
}

// ConfirmedDeletion requires the SDK's verified same-GUID deleted-object observation.
// A successful reply or an empty live-object search alone cannot authorize state removal.
func ConfirmedDeletion(result shared.ObjectMutationResult, err error) bool {
	if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified || result.PartialEffectsPossible ||
		result.TargetGUID == nil || *result.TargetGUID == (shared.GUID{}) || result.DeletionReadback == nil || !result.DeletionReadback.Complete {
		return false
	}
	for _, object := range result.DeletionReadback.Objects {
		guidValue, guidErr := object.Values("objectGUID").Single()
		deletedValue, deletedErr := object.Values("isDeleted").Single()
		if guidErr != nil || deletedErr != nil {
			continue
		}
		guid, guidErr := guidValue.GUID()
		deleted, deletedErr := deletedValue.Bool()
		if guidErr == nil && deletedErr == nil && guid == *result.TargetGUID && deleted {
			return true
		}
	}
	return false
}
