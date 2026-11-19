// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// applyProperties submits only changed typed fields and preserves accepted intent before projecting possibly hidden readback.
func (r *ServiceAccountResource) applyProperties(ctx context.Context, current, plan *ServiceAccountResourceModel, target shared.Identity, storage pendingData) diag.Diagnostics {
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
	result, err := r.client.UpdateServiceAccount(ctx, request)
	diagnostics.Append(recordMutation(current, result.Mutation, err)...)
	if result.Mutation.State == shared.ObjectWriteAccepted && result.Mutation.TargetGUID != nil && result.Mutation.TargetGUID.String() == current.ID.ValueString() {
		retainAcceptedProperties(current, plan, request)
	}
	if result.Account != nil {
		diagnostics.Append(mapState(current, result.Account)...)
	}
	if !diagnostics.HasError() {
		diagnostics.Append(r.refresh(ctx, current)...)
	}
	diagnostics.Append(finishPending(ctx, storage, pending, current, result.Mutation, err, !diagnostics.HasError())...)
	return diagnostics
}
