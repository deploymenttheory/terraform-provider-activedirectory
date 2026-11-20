// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const pendingKey = "ad_account_settings_pending_mutation"

type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// pendingMutation retains only nonsecret target/setting intent; a contradictory returned identity requires explicit recovery.
type pendingMutation struct {
	Version          int             `json:"version"`
	GUID             string          `json:"guid"`
	Operation        string          `json:"operation"`
	Flags            map[string]bool `json:"flags,omitempty"`
	ExpiresAt        string          `json:"expiresAt,omitempty"`
	ClearExpiration  bool            `json:"clearExpiration,omitempty"`
	IdentityConflict bool            `json:"identityConflict,omitempty"`
	ReportedGUID     string          `json:"reportedGuid,omitempty"`
}

// savePending must succeed before submission; unavailable durable storage never authorizes a write.
func savePending(ctx context.Context, storage pendingData, pending pendingMutation) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Account settings recovery storage unavailable", "No write was submitted because mutation intent cannot be retained.")}
	}
	pending.Version = 1
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Account settings recovery serialization failed", "No write was submitted because mutation intent cannot be retained.")}
	}
	return storage.SetKey(ctx, pendingKey, encoded)
}

// clearPending discards a rejected submission or a positively reconciled effect, never an uncertain observation.
func clearPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	if storage == nil {
		return nil
	}
	return storage.SetKey(ctx, pendingKey, nil)
}

// loadPending refuses malformed or future recovery state rather than treating it as permission to submit again.
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
	canonical, guidErr := identity.CanonicalGUID(pending.GUID)
	valid := err == nil && guidErr == nil && canonical == pending.GUID && pending.Version == 1
	if pending.ReportedGUID != "" {
		reported, reportedErr := identity.CanonicalGUID(pending.ReportedGUID)
		valid = valid && pending.IdentityConflict && reportedErr == nil && reported == pending.ReportedGUID
	}
	switch pending.Operation {
	case "flags":
		valid = valid && len(pending.Flags) > 0 && pending.ExpiresAt == "" && !pending.ClearExpiration
		model, patch := AccountSettingsResourceModel{}, accounts.AccountControlPatch{}
		allowed := map[string]bool{}
		for _, field := range flagProperties(&model, &patch) {
			allowed[field.name] = true
		}
		for field := range pending.Flags {
			valid = valid && allowed[field]
		}
	case "expiration":
		valid = valid && len(pending.Flags) == 0 && (pending.ClearExpiration != (pending.ExpiresAt != ""))
		if pending.ExpiresAt != "" {
			stamp, parseErr := time.Parse(time.RFC3339Nano, pending.ExpiresAt)
			valid = valid && parseErr == nil && stamp.UTC().Format(time.RFC3339Nano) == pending.ExpiresAt && stamp.Nanosecond()%100 == 0
		}
	default:
		valid = false
	}
	if !valid {
		diagnostics.AddError("Invalid account settings recovery state", "Stored intent cannot be interpreted. Independently reconcile the account before explicit GUID import.")
		return nil, diagnostics
	}
	return &pending, diagnostics
}

// pendingMatches requires positive complete values from the same GUID and cannot clear a contradictory mutation identity.
func pendingMatches(pending pendingMutation, values map[string]attr.Value) bool {
	id, ok := values["id"].(types.String)
	if !ok || id.ValueString() != pending.GUID || pending.IdentityConflict {
		return false
	}
	if pending.Operation == "flags" {
		for name, wanted := range pending.Flags {
			value, visible := observedFlag(values, name)
			if !visible || value != wanted {
				return false
			}
		}
		return len(pending.Flags) > 0
	}
	if pending.ClearExpiration {
		never, visible := observedFlag(values, "never_expires")
		return visible && never
	}
	observed, ok := values["expires_at"].(types.String)
	return ok && !observed.IsNull() && !observed.IsUnknown() && observed.ValueString() == pending.ExpiresAt
}

// reconcilePending is read-only and blocks later writes until the previous intent is proved on the selected account.
func reconcilePending(ctx context.Context, storage pendingData, guid string, values map[string]attr.Value) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	if pending.GUID != guid || !pendingMatches(*pending, values) {
		diagnostics.AddError(
			"Account settings write needs reconciliation",
			"Prior submitted intent is not positively observable on the same GUID. No new write is permitted. Independently verify prior effects before explicit GUID import.",
		)
		return diagnostics
	}
	diagnostics.Append(clearPending(ctx, storage)...)
	return diagnostics
}

// mutationIdentityConflict prevents a positive observation of one account from concealing a contradictory write target.
func mutationIdentityConflict(result shared.ObjectMutationResult, guid string) (string, bool) {
	if result.TargetGUID != nil && result.TargetGUID.String() != guid {
		return result.TargetGUID.String(), true
	}
	for _, object := range []*shared.Object{result.Before, result.After} {
		if object == nil || object.Values("objectGUID").State != shared.AttributePresent {
			continue
		}
		value, err := object.Values("objectGUID").Single()
		returned, parseErr := value.GUID()
		if err != nil || parseErr != nil {
			return "", true
		}
		if returned.String() != guid {
			return returned.String(), true
		}
	}
	return "", false
}
