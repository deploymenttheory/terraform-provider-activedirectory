// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Invoke submits one SDK unlock and reports its observed outcome without retry, enablement changes or convergence claims.
func (a *AccountUnlockAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var model AccountUnlockActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	target, diagnostics := constructTarget(model)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if a.client == nil || a.capabilities.RequireADDS(ActionName) != nil {
		resp.Diagnostics.AddError("Account unlock unavailable", "Configure a qualified AD DS account service before invoking this action.")
		return
	}
	if !strings.EqualFold(model.Server.ValueString(), a.server) {
		resp.Diagnostics.AddError("Account unlock server differs", "server does not match the configured session's observed DC. No unlock was submitted.")
		return
	}
	if ctx.Err() != nil {
		resp.Diagnostics.AddError("Account unlock canceled", "The request was canceled before submission. No unlock was attempted.")
		return
	}
	progress(resp, "Submitting one account unlock on the explicitly selected DC.")
	result, err := a.client.UnlockAccount(ctx, target, shared.ObjectMutationOptions{})
	outcome := lifecycle.ObserveMutation(types.StringValue(target.String()), result.Mutation, err)
	resp.Diagnostics.Append(outcome.Diagnostics...)
	if resp.Diagnostics.HasError() {
		if outcome.ReconcileRequired {
			resp.Diagnostics.AddError("Account unlock requires reconciliation",
				"Target GUID: "+target.String()+"; selected DC: "+a.server+". The submission may have effects. Verify lockoutTime on that DC before explicitly invoking again.")
		}
		return
	}
	if result.Account != nil && result.Account.LockoutTime != nil && *result.Account.LockoutTime != 0 {
		resp.Diagnostics.AddError(
			"Account unlock readback differs",
			"The selected DC still returned a nonzero lockoutTime. Verify the account independently; no retry was performed.",
		)
		return
	}
	verified := result.Mutation.Verified && !outcome.ReconcileRequired && result.Account != nil && result.Account.LockoutTime != nil && *result.Account.LockoutTime == 0
	if verified {
		value, valueErr := result.Account.Raw.Values("objectGUID").Single()
		guid, guidErr := value.GUID()
		verified = valueErr == nil && guidErr == nil && guid.String() == target.String()
		if valueErr == nil && guidErr == nil && guid.String() != target.String() {
			resp.Diagnostics.AddError(
				"Account unlock readback identity differs",
				"The typed readback belongs to another GUID. Preserve the requested identity and reconcile independently.",
			)
			return
		}
	}
	if !verified {
		resp.Diagnostics.AddWarning(
			"Account unlock accepted without complete readback",
			"The server accepted the unlock, but a same-GUID lockoutTime=0 observation is unavailable. "+
				"Verify the selected DC independently; do not infer current login eligibility or forest convergence.",
		)
		progress(resp, "Unlock submission accepted; its effect requires independent verification on the selected DC.")
		return
	}
	progress(resp, "The same account GUID has lockoutTime=0 on the selected DC. This action does not enable disabled accounts or prove forest-wide convergence.")
}

// progress allows focused Framework fixtures to omit the event callback without changing invocation semantics.
func progress(resp *action.InvokeResponse, message string) {
	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: message})
	}
}
