// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

// applySettings positively adopts the existing GUID before any write and validates every configured phase against visible observations.
func (r *AccountSettingsResource) applySettings(ctx context.Context, model *AccountSettingsResourceModel, intent settingsIntent, storage pendingData) (bool, diag.Diagnostics) {
	target, _ := shared.IdentityByGUID(intent.GUID)
	snapshot, err := common.Read(ctx, r.client, target)
	var diagnostics diag.Diagnostics
	if err != nil {
		diagnostics.Append(snapshot.Diagnostics...)
		diagnostics.AddError(
			"Existing account adoption failed",
			"The selected account could not be observed by GUID. No settings write was submitted; an account is never created implicitly.",
		)
		return false, diagnostics
	}
	diagnostics.Append(mapSnapshot(ctx, model, snapshot)...)
	if diagnostics.HasError() {
		return false, diagnostics
	}
	diagnostics.Append(reconcilePending(ctx, storage, intent.GUID, snapshot.Values)...)
	diagnostics.Append(validateObservedIntent(intent, snapshot)...)
	if diagnostics.HasError() {
		return true, diagnostics
	}
	patch, changes := changedFlags(intent, snapshot)
	if len(changes) > 0 {
		pending := pendingMutation{GUID: intent.GUID, Operation: "flags", Flags: changes}
		diagnostics.Append(savePending(ctx, storage, pending)...)
		if diagnostics.HasError() {
			return true, diagnostics
		}
		result, writeErr := r.client.UpdateAccountControl(ctx, accounts.AccountControlRequest{Identity: target, Patch: patch})
		fallback := snapshot
		if result.Account != nil {
			fallback.Control.Account = result.Account
			fallback.Values, fallback.Diagnostics = common.Project(*result.Account, *snapshot.Expiration.Expiration)
		}
		var phaseDiagnostics diag.Diagnostics
		snapshot, phaseDiagnostics = r.finishPhase(ctx, model, storage, pending, result.Mutation, writeErr, fallback)
		diagnostics.Append(phaseDiagnostics...)
		if diagnostics.HasError() {
			return true, diagnostics
		}
	}
	diagnostics.Append(validateObservedIntent(intent, snapshot)...)
	if !intentFlagsMatch(intent, snapshot) {
		diagnostics.AddError(
			"Configured account flags changed during apply",
			"Earlier accepted changes are retained. Configured flags no longer match on the same account; no later expiration phase was submitted.",
		)
	}
	if diagnostics.HasError() {
		return true, diagnostics
	}
	if !expirationChangeRequired(intent, snapshot) {
		return true, diagnostics
	}
	pending := pendingMutation{GUID: intent.GUID, Operation: "expiration", ClearExpiration: intent.ClearExpiration}
	if intent.ExpiresAt != nil {
		pending.ExpiresAt = intent.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	diagnostics.Append(savePending(ctx, storage, pending)...)
	if diagnostics.HasError() {
		return true, diagnostics
	}
	var result accounts.AccountExpirationMutationResult
	if intent.ClearExpiration {
		result, err = r.client.ClearAccountExpiration(ctx, target, shared.ObjectMutationOptions{})
	} else {
		result, err = r.client.SetAccountExpiration(ctx, accounts.SetAccountExpirationRequest{Identity: target, At: *intent.ExpiresAt})
	}
	fallback := snapshot
	if result.Expiration != nil {
		fallback.Expiration.Expiration = result.Expiration
		fallback.Values, fallback.Diagnostics = common.Project(*snapshot.Control.Account, *result.Expiration)
	}
	refreshed, phaseDiagnostics := r.finishPhase(ctx, model, storage, pending, result.Mutation, err, fallback)
	diagnostics.Append(phaseDiagnostics...)
	if !diagnostics.HasError() && !intentFlagsMatch(intent, refreshed) {
		diagnostics.AddError(
			"Configured account flags changed during expiration update",
			"The accepted expiration effect is retained. Independently observe drift before another apply; no automatic rollback or flag replay was performed.",
		)
	}
	return true, diagnostics
}

// validateObservedIntent refuses hidden configured fields and cross-field conflicts before any phase is submitted.
func validateObservedIntent(intent settingsIntent, snapshot common.Snapshot) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	for name := range intent.Flags {
		if _, visible := observedFlag(snapshot.Values, name); !visible {
			diagnostics.AddError("Configured account setting is unavailable", name+" was not positively observed; no later write is authorized from hidden state.")
		}
	}
	if intent.ClearExpiration || intent.ExpiresAt != nil {
		if _, visible := observedFlag(snapshot.Values, "never_expires"); !visible || snapshot.Expiration.Expiration == nil || snapshot.Expiration.Expiration.Value == nil {
			diagnostics.AddError("Account expiration is unavailable", "A configured expiration requires visible current expiration metadata before any write phase.")
		}
	}
	if change, configured := intent.Flags["change_password_at_logon"]; configured && change {
		never, visible := observedFlag(snapshot.Values, "password_never_expires")
		if wanted, owned := intent.Flags["password_never_expires"]; owned {
			never, visible = wanted, true
		}
		if !visible || never {
			diagnostics.AddError(
				"Password settings conflict with account state",
				"Requiring a password change needs a visible false password_never_expires setting, or an explicit false in the same atomic patch.",
			)
		}
	}
	return diagnostics
}

// changedFlags constructs only configured differences, preserving unrelated or concurrent unowned settings in the SDK atomic patch.
func changedFlags(intent settingsIntent, snapshot common.Snapshot) (accounts.AccountControlPatch, map[string]bool) {
	patch := accounts.AccountControlPatch{}
	model := AccountSettingsResourceModel{}
	changes := map[string]bool{}
	for _, field := range flagProperties(&model, &patch) {
		wanted, owned := intent.Flags[field.name]
		current, visible := observedFlag(snapshot.Values, field.name)
		if !owned || !visible || wanted == current {
			continue
		}
		value := wanted
		*field.target = &value
		changes[field.name] = wanted
	}
	return patch, changes
}

// intentFlagsMatch verifies configured fields only, never adopting an observed unconfigured value as future write intent.
func intentFlagsMatch(intent settingsIntent, snapshot common.Snapshot) bool {
	for name, wanted := range intent.Flags {
		current, visible := observedFlag(snapshot.Values, name)
		if !visible || current != wanted {
			return false
		}
	}
	return true
}

// expirationChangeRequired keeps omission and already-matching native never-expire representations free of writes.
func expirationChangeRequired(intent settingsIntent, snapshot common.Snapshot) bool {
	if intent.ClearExpiration {
		never, visible := observedFlag(snapshot.Values, "never_expires")
		return !visible || !never
	}
	if intent.ExpiresAt == nil {
		return false
	}
	pending := pendingMutation{GUID: intent.GUID, Operation: "expiration", ExpiresAt: intent.ExpiresAt.UTC().Format(time.RFC3339Nano)}
	return !pendingMatches(pending, snapshot.Values)
}
