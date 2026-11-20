// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

const pendingKey = "ad_computer_service_account_pending_mutation"

type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

type pendingMutation struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Present   bool   `json:"present"`
	Conflict  string `json:"conflict,omitempty"`
}

// savePending durably retains the exact nonsecret edge intent before a write can be submitted.
func savePending(ctx context.Context, storage pendingData, pending pendingMutation) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Association recovery storage unavailable", "No write was submitted because its intent cannot be retained.")}
	}
	pending.Version = 1
	pending.Operation = "remove"
	if pending.Present {
		pending.Operation = "add"
	}
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Association recovery serialization failed", "No write was submitted.")}
	}
	return storage.SetKey(ctx, pendingKey, encoded)
}

// loadPending rejects malformed or future recovery data rather than authorizing another mutation.
func loadPending(ctx context.Context, storage pendingData) (*pendingMutation, diag.Diagnostics) {
	if storage == nil {
		return nil, nil
	}
	encoded, diagnostics := storage.GetKey(ctx, pendingKey)
	if diagnostics.HasError() || len(encoded) == 0 {
		return nil, diagnostics
	}
	var pending pendingMutation
	err := json.Unmarshal(encoded, &pending)
	computer, account, validation := identity.ParseRelationshipID(pending.ID)
	canonical, _ := identity.RelationshipID(computer, account)
	validOperation := (pending.Operation == "add" && pending.Present) || (pending.Operation == "remove" && !pending.Present)
	if err != nil || pending.Version != 1 || validation.HasError() || canonical != pending.ID || !validOperation {
		diagnostics.AddError("Invalid association recovery state", "Independently reconcile the exact link before explicit composite-ID import.")
		return nil, diagnostics
	}
	return &pending, diagnostics
}

// clearPending clears only a known rejection or a fresh, positively reconciled effect.
func clearPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	if storage == nil {
		return nil
	}
	return storage.SetKey(ctx, pendingKey, nil)
}

// reconcilePending blocks a later write or tainted replacement until the prior edge intent is observed.
func reconcilePending(ctx context.Context, storage pendingData, id string, edge common.Edge) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	if pending.ID != id || pending.Conflict != "" || !edge.Confirmed || edge.Present != pending.Present {
		diagnostics.AddError(
			"Association write needs reconciliation",
			"The original pair and mutation intent are retained. No mutation or replacement is authorized from an incomplete or conflicting observation.",
		)
		return diagnostics
	}
	diagnostics.Append(clearPending(ctx, storage)...)
	return diagnostics
}
