// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// change submits one atomic linked-value modification and reconciles through a fresh assertion without retrying.
func (r *GroupMemberResource) change(
	ctx context.Context, model *GroupMemberResourceModel, storage pendingData, group, member shared.Identity, id string, present bool,
) (bool, diag.Diagnostics) {
	pending := pendingMutation{ID: id, Present: present}
	diagnostics := savePending(ctx, storage, pending)
	if diagnostics.HasError() {
		return false, diagnostics
	}
	var result memberships.GroupMembershipMutationResult
	var writeErr error
	if present {
		result, writeErr = r.client.AddGroupMembers(ctx, constructAdd(group, member))
	} else {
		result, writeErr = r.client.RemoveGroupMembers(ctx, constructRemove(group, member))
	}
	mutation := result.Mutation
	mayHaveEffect := mutation.State == shared.ObjectWriteAccepted || mutation.State == shared.ObjectWriteUncertain ||
		mutation.PartialEffectsPossible || shared.IsUncertain(writeErr)
	if !mayHaveEffect {
		diagnostics.Append(clearPending(ctx, storage)...)
		diagnostics.AddError("Membership mutation rejected", "The direct-link change was not accepted. Other members were not modified; peer error text is withheld.")
		return false, diagnostics
	}
	model.IsMember = types.BoolNull()
	if conflict := mutationConflict(result, group, member); conflict != "" {
		pending.Conflict = conflict
		diagnostics.Append(savePending(ctx, storage, pending)...)
		diagnostics.AddError(
			"Conflicting membership mutation identity",
			"The result disagrees with the submitted GUID pair. Original identity and recovery intent are retained for independent reconciliation.",
		)
		return true, diagnostics
	}
	// A successful atomic value deletion with critical permissive handling establishes that exact link's absence at acknowledgement.
	// This is operation-specific proof; an uncertain write or any failed readback cannot take this path.
	if !present && mutation.State == shared.ObjectWriteAccepted && writeErr == nil && mutation.TargetGUID != nil {
		model.IsMember = types.BoolValue(false)
		diagnostics.Append(clearPending(ctx, storage)...)
		return true, diagnostics
	}
	edge, _ := common.Read(ctx, r.client, group, member)
	if edge.Confirmed && edge.Present == present {
		diagnostics.Append(clearPending(ctx, storage)...)
		if present {
			mapState(model, edge)
		}
		if writeErr != nil || mutation.State != shared.ObjectWriteAccepted {
			diagnostics.AddWarning("Membership effect independently reconciled", "Fresh directory observations match the exact submitted link intent. No write was replayed.")
		}
		return true, diagnostics
	}
	diagnostics.AddError(
		"Membership write needs reconciliation",
		"The submitted change may have effects, but a fresh observation did not confirm the same pair. Known identity and intent are retained; no mutation was retried.",
	)
	return true, diagnostics
}

// mutationConflict preserves contradictory positive identity evidence instead of allowing a later matching read to erase it.
func mutationConflict(result memberships.GroupMembershipMutationResult, group, member shared.Identity) string {
	mutation := result.Mutation
	if mutation.TargetGUID != nil && mutation.TargetGUID.String() != group.String() {
		return mutation.TargetGUID.String()
	}
	for _, object := range []*shared.Object{mutation.Before, mutation.After} {
		if object == nil {
			continue
		}
		value, err := object.Values("objectGUID").Single()
		guid, parseErr := value.GUID()
		if err == nil && parseErr == nil && guid.String() != group.String() {
			return guid.String()
		}
	}
	if len(result.Members) != 1 || result.Members[0].Identity.Form() != shared.IdentityFormGUID || result.Members[0].Identity.String() != member.String() {
		return "member identity differs"
	}
	return ""
}
