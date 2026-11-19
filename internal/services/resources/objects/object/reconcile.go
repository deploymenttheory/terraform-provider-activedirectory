// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

const (
	pendingKey = "ad_object_pending_mutation"
	ackKey     = "ad_object_acknowledged_observations"
)

type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// pendingMutation stores recovery identity and hashed raw intent, never raw attribute values or provider credentials.
type pendingMutation struct {
	Version          int             `json:"version"`
	Operation        string          `json:"operation"`
	GUID             string          `json:"guid,omitempty"`
	Name             string          `json:"name,omitempty"`
	ParentDN         string          `json:"parentDn,omitempty"`
	Changes          []pendingChange `json:"changes,omitempty"`
	Acknowledged     bool            `json:"acknowledged,omitempty"`
	IdentityConflict bool            `json:"identityConflict,omitempty"`
}

type pendingChange struct {
	Attribute string                     `json:"attribute"`
	Kind      shared.AttributeChangeKind `json:"kind"`
	Hashes    []string                   `json:"hashes,omitempty"`
}

// acknowledgedObservation connects one accepted intent with its exact raw readback, without claiming independent verification.
type acknowledgedObservation struct {
	GUID            string `json:"guid"`
	IntentHash      string `json:"intentHash"`
	ObservationHash string `json:"observationHash"`
}

type acknowledgements map[string]acknowledgedObservation

// eligibleAcknowledgements prevents relinquished ownership from authorizing a later re-acquisition through stale accepted intent.
func eligibleAcknowledgements(model *ObjectResourceModel, stored acknowledgements) acknowledgements {
	result := acknowledgements{}
	changes, _, diagnostics := constructPlan(model, false)
	if diagnostics.HasError() {
		return result
	}
	for _, change := range changes {
		hashed := hashedChange(change)
		key := changeKey(hashed)
		if ack, found := stored[key]; found && ack.IntentHash == intentHash(hashed) {
			result[key] = ack
		}
	}
	return result
}

// savePending must succeed before submission so a later transport failure cannot authorize blind replay.
func savePending(ctx context.Context, storage pendingData, pending pendingMutation) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Object recovery storage unavailable", "Mutation intent cannot be retained. No write was submitted.")}
	}
	pending.Version = 1
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Object recovery serialization failed", "No write was submitted.")}
	}
	return storage.SetKey(ctx, pendingKey, encoded)
}

// clearPending retires only a proven or definitely rejected operation.
func clearPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	if storage == nil {
		return nil
	}
	return storage.SetKey(ctx, pendingKey, nil)
}

// loadPending rejects unreadable recovery records before allowing another mutation.
func loadPending(ctx context.Context, storage pendingData) (*pendingMutation, diag.Diagnostics) {
	if storage == nil {
		return nil, nil
	}
	encoded, diagnostics := storage.GetKey(ctx, pendingKey)
	if diagnostics.HasError() || len(encoded) == 0 {
		return nil, diagnostics
	}
	var pending pendingMutation
	if err := json.Unmarshal(encoded, &pending); err != nil || pending.Version != 1 {
		diagnostics.AddError("Invalid object recovery state", "Stored intent cannot be read. Reconcile independently before explicit GUID recovery.")
		return nil, diagnostics
	}
	switch pending.Operation {
	case "create", "rename", "move", "update", "delete":
		return &pending, diagnostics
	default:
		diagnostics.AddError("Invalid object recovery operation", "No further mutation is authorized from an unknown stored operation.")
		return nil, diagnostics
	}
}

// guardPending blocks replay and tainted-resource replacement until prior effects have been reconciled.
func guardPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if pending != nil {
		diagnostics.AddError("Object mutation needs reconciliation", "Earlier intent remains unresolved. Verify its effects before another write or explicit GUID recovery.")
	}
	return diagnostics
}

// loadAcknowledgements retains accepted normalization/clear observations separately from independent verification.
func loadAcknowledgements(ctx context.Context, storage pendingData) (acknowledgements, diag.Diagnostics) {
	result := acknowledgements{}
	if storage == nil {
		return result, nil
	}
	encoded, diagnostics := storage.GetKey(ctx, ackKey)
	if !diagnostics.HasError() && len(encoded) != 0 {
		if err := json.Unmarshal(encoded, &result); err != nil || result == nil {
			diagnostics.AddError("Invalid object acknowledgement state", "Accepted observations cannot be read. No subsequent mutation is permitted until recovery.")
		}
	}
	return result, diagnostics
}

// saveAcknowledgements records only fingerprints for changes the SDK acknowledged and observed on this same GUID.
func saveAcknowledgements(ctx context.Context, storage pendingData, acknowledged acknowledgements) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Object acknowledgement storage unavailable", "Preserve the known GUID and mutation intent before recovery.")}
	}
	encoded, err := json.Marshal(acknowledged)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Object acknowledgement serialization failed", "Known identity and pending intent are retained.")}
	}
	return storage.SetKey(ctx, ackKey, encoded)
}

// changeKey includes the operation so authoritative ownership cannot reuse a value-level acknowledgement.
func changeKey(change pendingChange) string {
	return fmt.Sprintf("%s/%d", change.Attribute, change.Kind)
}

// digest hashes binary values without logging or serializing their contents into recovery records.
func digest(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

// hashedChange preserves enough intent to compare positive observations after an uncertain write.
func hashedChange(change shared.AttributeChange) pendingChange {
	result := pendingChange{Attribute: strings.ToLower(change.Attribute), Kind: change.Kind}
	for _, value := range change.Values {
		result.Hashes = append(result.Hashes, digest(value.Bytes()))
	}
	slices.Sort(result.Hashes)
	return result
}

// intentHash separates empty clears from one zero-length raw value.
func intentHash(change pendingChange) string {
	return digest([]byte(changeKey(change) + "\x00" + strings.Join(change.Hashes, ",")))
}

// observationHash includes visibility state so an unloaded/partial result cannot match an earlier accepted observation.
func observationHash(object shared.Object, attribute string) string {
	values := object.Values(attribute)
	return digest([]byte(fmt.Sprintf("%d/%d\x00%s", values.State, len(values.Values), strings.Join(common.EncodeValues(values.Values), ","))))
}

// acknowledgementMatches is evidence of an unchanged accepted observation, not proof inferred from omitted values.
func acknowledgementMatches(acks acknowledgements, change shared.AttributeChange, object shared.Object) bool {
	intent := hashedChange(change)
	ack, found := acks[changeKey(intent)]
	guid, valid := common.GUID(object)
	return found && valid && ack.GUID == guid && ack.IntentHash == intentHash(intent) && ack.ObservationHash == observationHash(object, change.Attribute)
}

// finishPending retains all accepted/uncertain failures; a definite rejection without possible effects permits a later corrected request.
func finishPending(ctx context.Context, storage pendingData, pending pendingMutation, result shared.ObjectMutationResult, err error, success bool) diag.Diagnostics {
	if success || (result.State == shared.ObjectWriteNotAttempted || result.State == shared.ObjectWriteRejected) && !result.PartialEffectsPossible && !shared.IsUncertain(err) {
		return clearPending(ctx, storage)
	}
	return savePending(ctx, storage, pending)
}

// reconcilePending uses positive same-GUID observations. Failed creation stays blocked because Terraform taints it.
func reconcilePending(ctx context.Context, storage pendingData, object shared.Object, acks acknowledgements) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	guid, valid := common.GUID(object)
	matched := valid && guid == pending.GUID && !pending.IdentityConflict && pending.Operation != "create" && pending.Operation != "delete"
	if pending.Name != "" {
		value, err := object.Values("name").Single()
		matched = matched && err == nil && string(value.Bytes()) == pending.Name
	}
	if pending.ParentDN != "" {
		dn := object.DistinguishedName
		if value, err := object.Values("distinguishedName").Single(); err == nil {
			dn = string(value.Bytes())
		}
		parent, valid := identity.ParentDN(dn)
		matched = matched && valid && strings.EqualFold(parent, pending.ParentDN)
	}
	for _, change := range pending.Changes {
		ack, found := acks[changeKey(change)]
		accepted := pending.Acknowledged && found && ack.GUID == guid && ack.IntentHash == intentHash(change) && ack.ObservationHash == observationHash(object, change.Attribute)
		matched = matched && (accepted || positiveEffect(change, object))
	}
	if matched {
		diagnostics.Append(clearPending(ctx, storage)...)
	} else {
		diagnostics.AddError("Object mutation still unresolved", "Readback did not establish the earlier effect. Identity and intent remain; no replay is permitted.")
	}
	return diagnostics
}

// positiveEffect never proves uncertain removal/clear from omission or byte-different DN representations.
func positiveEffect(change pendingChange, object shared.Object) bool {
	values := object.Values(change.Attribute)
	if values.State != shared.AttributePresent {
		return false
	}
	observed := make([]string, 0, len(values.Values))
	for _, value := range values.Values {
		observed = append(observed, digest(value.Bytes()))
	}
	slices.Sort(observed)
	switch change.Kind {
	case shared.AttributeSet:
		return slices.Equal(observed, change.Hashes)
	case shared.AttributeAddValues:
		for _, hash := range change.Hashes {
			if !slices.Contains(observed, hash) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
