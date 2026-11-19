// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// applyProperties submits only changed typed fields and preserves accepted intent before projecting possibly hidden readback.
func (r *UserResource) applyProperties(ctx context.Context, current, plan *UserResourceModel, target shared.Identity, storage pendingData) diag.Diagnostics {
	request, changed := constructUpdate(current, plan, target)
	if !changed {
		return nil
	}
	pending := pendingForUpdate(request, plan)
	pending.GUID = current.ID.ValueString()
	diagnostics := savePending(ctx, storage, pending)
	if diagnostics.HasError() {
		return diagnostics
	}
	result, err := r.client.UpdateUser(ctx, request)
	diagnostics.Append(recordMutation(current, result.Mutation, err)...)
	if result.Mutation.State == shared.ObjectWriteAccepted && result.Mutation.TargetGUID != nil && result.Mutation.TargetGUID.String() == current.ID.ValueString() {
		retainAcceptedProperties(current, plan, request)
	}
	if result.User != nil {
		diagnostics.Append(mapState(current, result.User)...)
	}
	if !diagnostics.HasError() {
		diagnostics.Append(r.refresh(ctx, current)...)
	}
	diagnostics.Append(finishPending(ctx, storage, pending, current, result.Mutation, err, !diagnostics.HasError())...)
	return diagnostics
}

// resetPassword acknowledges only the selected GUID's accepted operation. A pending reset stores its version, never a secret or hash.
func (r *UserResource) resetPassword(
	ctx context.Context,
	current *UserResourceModel,
	version int64,
	password types.String,
	target shared.Identity,
	storage pendingData,
) diag.Diagnostics {
	pending := pendingMutation{Operation: "password", GUID: current.ID.ValueString(), PasswordVersion: version}
	diagnostics := savePending(ctx, storage, pending)
	if diagnostics.HasError() {
		return diagnostics
	}
	buffer := []byte(password.ValueString())
	defer clear(buffer)
	result, err := r.client.ResetAccountPassword(ctx, accounts.PasswordResetRequest{Identity: target, NewPassword: buffer})
	accepted := result.State == shared.ObjectWriteAccepted && result.TargetGUID != nil && result.TargetGUID.String() == current.ID.ValueString() && !shared.IsUncertain(err)
	if accepted {
		current.PasswordWOVersion = types.Int64Value(version)
	}
	if accepted && err == nil {
		diagnostics.Append(clearPending(ctx, storage)...)
		return diagnostics
	}
	if (result.State == shared.ObjectWriteRejected || result.State == shared.ObjectWriteNotAttempted) && !shared.IsUncertain(err) {
		diagnostics.Append(clearPending(ctx, storage)...)
	}
	diagnostics.AddError(
		"User password change did not complete",
		"Known GUID and acknowledged version are retained. Uncertain password effects cannot be verified through secret readback; "+
			"reconcile independently before explicit recovery or another reset.",
	)
	return diagnostics
}
