// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// change submits one atomic linked-value modification and reconciles through a fresh assertion without retrying.
func (r *ComputerServiceAccountResource) change(
	ctx context.Context, model *ComputerServiceAccountResourceModel, storage pendingData, computer, account shared.Identity, id string, present bool,
) (bool, diag.Diagnostics) {
	pending := pendingMutation{ID: id, Present: present}
	diagnostics := savePending(ctx, storage, pending)
	if diagnostics.HasError() {
		return false, diagnostics
	}
	var result sdkaccounts.ComputerServiceAccountMutationResult
	var writeErr error
	if present {
		result, writeErr = r.client.AddComputerServiceAccounts(ctx, constructAdd(computer, account))
	} else {
		result, writeErr = r.client.RemoveComputerServiceAccounts(ctx, constructRemove(computer, account))
	}
	mutation := result.Mutation
	mayHaveEffect := mutation.State == shared.ObjectWriteAccepted || mutation.State == shared.ObjectWriteUncertain ||
		mutation.PartialEffectsPossible || shared.IsUncertain(writeErr)
	if !mayHaveEffect {
		diagnostics.Append(clearPending(ctx, storage)...)
		diagnostics.AddError("Association mutation rejected", "The direct-link change was not accepted. Other accounts were not modified; peer error text is withheld.")
		return false, diagnostics
	}
	model.IsAssociated = types.BoolNull()
	if conflict := mutationConflict(result, computer, account); conflict != "" {
		pending.Conflict = conflict
		diagnostics.Append(savePending(ctx, storage, pending)...)
		diagnostics.AddError(
			"Conflicting association mutation identity",
			"The result disagrees with the submitted GUID pair. Original identity and recovery intent are retained for independent reconciliation.",
		)
		return true, diagnostics
	}
	// The SDK distinguishes acknowledged value deletion from verification of the resulting collection.
	// An omitted final attribute is not empty-collection proof; the exact atomic deletion acknowledgement establishes this pair's absence.
	if !present && removalAcknowledged(result, writeErr) {
		model.IsAssociated = types.BoolValue(false)
		diagnostics.Append(clearPending(ctx, storage)...)
		if writeErr != nil {
			diagnostics.AddWarning(
				"Association removal acknowledged",
				"The server acknowledged the exact value deletion. The final collection attribute was omitted, so collection visibility remains indeterminate.",
			)
		}
		return true, diagnostics
	}
	edge, _ := common.Read(ctx, r.client, computer, account)
	if edge.Confirmed && edge.Present == present {
		diagnostics.Append(clearPending(ctx, storage)...)
		if present {
			mapState(model, edge)
		}
		if writeErr != nil || mutation.State != shared.ObjectWriteAccepted {
			diagnostics.AddWarning("Association effect independently reconciled", "Fresh directory observations match the exact submitted link intent. No write was replayed.")
		}
		return true, diagnostics
	}
	diagnostics.AddError(
		"Association write needs reconciliation",
		"The submitted change may have effects, but a fresh observation did not confirm the same pair. Known identity and intent are retained; no mutation was retried.",
	)
	return true, diagnostics
}

// mutationConflict preserves contradictory positive identity evidence instead of allowing a later matching read to erase it.
func mutationConflict(result sdkaccounts.ComputerServiceAccountMutationResult, computer, account shared.Identity) string {
	mutation := result.Mutation
	if mutation.TargetGUID != nil && mutation.TargetGUID.String() != computer.String() {
		return mutation.TargetGUID.String()
	}
	for _, object := range []*shared.Object{mutation.Before, mutation.After} {
		if object == nil {
			continue
		}
		value, err := object.Values("objectGUID").Single()
		guid, parseErr := value.GUID()
		if err == nil && parseErr == nil && guid.String() != computer.String() {
			return guid.String()
		}
	}
	if result.Links != nil && result.Links.Version != nil && result.Links.Version.GUID.String() != computer.String() {
		return result.Links.Version.GUID.String()
	}
	if len(result.Accounts) != 1 || !accountObservationMatches(result.Accounts[0], account) {
		return "service account identity differs"
	}
	return ""
}

// removalAcknowledged consumes the SDK's operation acknowledgement, never inferring deletion from omitted attributes or mixed failures.
func removalAcknowledged(result sdkaccounts.ComputerServiceAccountMutationResult, writeErr error) bool {
	mutation := result.Mutation
	if mutation.State != shared.ObjectWriteAccepted || mutation.TargetGUID == nil || mutation.PartialEffectsPossible {
		return false
	}
	if writeErr == nil {
		return true
	}
	// The SDK deliberately returns this exact sentinel after a successful deletion when the final attribute is omitted.
	// Wrapped, joined, transport, range and resolution errors remain unresolved rather than being treated as this outcome.
	if !errors.Is(writeErr, shared.ErrObjectVerification) || errors.Unwrap(writeErr) != nil || result.Links == nil {
		return false
	}
	if _, joined := writeErr.(interface{ Unwrap() []error }); joined {
		return false
	}
	links := result.Links
	return links.Version != nil && links.Version.GUID == *mutation.TargetGUID && links.Complete && !links.AttributeReturned &&
		len(links.Links) == 0 && len(links.Ranges) == 1 && links.Ranges[0].Error == nil
}

// accountObservationMatches requires the SDK account-resolution result to agree with the requested GUID.
func accountObservationMatches(result shared.IdentityResult, target shared.Identity) bool {
	if !result.Complete || result.Object == nil || len(result.Candidates) > 1 {
		return false
	}
	objects := append([]shared.Object{*result.Object}, result.Candidates...)
	for _, object := range objects {
		value, err := object.Values("objectGUID").Single()
		guid, parseErr := value.GUID()
		if err != nil || parseErr != nil || guid.String() != target.String() {
			return false
		}
	}
	return true
}
