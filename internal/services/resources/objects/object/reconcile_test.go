// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type memoryPrivate map[string][]byte

func (m memoryPrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return m[key], nil
}

func (m memoryPrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	m[key] = value
	return nil
}

// TestRecoveryHashesDoNotExposeRawValues preserves uncertain intent without duplicating raw values into private records.
func TestRecoveryHashesDoNotExposeRawValues(t *testing.T) {
	t.Parallel()
	storage := memoryPrivate{}
	secret := []byte("distinctive-raw-value-for-private-storage-test")
	change := shared.AttributeChange{Attribute: "description", Kind: shared.AttributeSet, Values: []shared.RawValue{shared.NewRawValue(secret)}}
	pending := mutationCheckpoint("update", fixtureGUID, []shared.AttributeChange{change})
	if diagnostics := savePending(t.Context(), storage, pending); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	encoded := string(storage[pendingKey])
	if strings.Contains(encoded, string(secret)) || strings.Contains(encoded, base64.StdEncoding.EncodeToString(secret)) {
		t.Fatal("private recovery state retained raw attribute contents")
	}
	if !guardPending(t.Context(), storage).HasError() {
		t.Fatal("stored uncertain intent permitted another write")
	}
}

// TestOmissionCannotReconcileUncertainRemoval keeps clear/remove checkpoints even when the attribute is not returned.
func TestOmissionCannotReconcileUncertainRemoval(t *testing.T) {
	t.Parallel()
	directory := newUnitDirectory()
	directory.name, directory.parent, directory.class = "fixture", "OU=first,DC=unit,DC=invalid", "contact"
	object := directory.snapshot([]string{"description"})
	for _, kind := range []shared.AttributeChangeKind{shared.AttributeClear, shared.AttributeRemoveValues} {
		storage := memoryPrivate{}
		change := shared.AttributeChange{Attribute: "description", Kind: kind}
		if kind == shared.AttributeRemoveValues {
			change.Values = []shared.RawValue{shared.NewRawValue([]byte("removed"))}
		}
		if savePending(t.Context(), storage, mutationCheckpoint("update", fixtureGUID, []shared.AttributeChange{change})).HasError() {
			t.Fatal("pending fixture failed")
		}
		if !reconcilePending(t.Context(), storage, object, nil).HasError() || !guardPending(t.Context(), storage).HasError() {
			t.Fatal("an omitted raw attribute proved uncertain removal or allowed replay")
		}
	}
}

// TestAcknowledgementsBindGUIDAndVisibility distinguishes accepted unchanged readback from an empty or unrelated observation.
func TestAcknowledgementsBindGUIDAndVisibility(t *testing.T) {
	t.Parallel()
	directory := newUnitDirectory()
	directory.name, directory.parent, directory.class = "fixture", "OU=first,DC=unit,DC=invalid", "contact"
	object := directory.snapshot([]string{"description"})
	change := shared.AttributeChange{Attribute: "description", Kind: shared.AttributeClear}
	intent := hashedChange(change)
	acks := acknowledgements{changeKey(intent): {GUID: fixtureGUID, IntentHash: intentHash(intent), ObservationHash: observationHash(object, "description")}}
	if !acknowledgementMatches(acks, change, object) {
		t.Fatal("a known accepted unchanged observation was lost")
	}
	object.RequestedProperties = nil
	if acknowledgementMatches(acks, change, object) {
		t.Fatal("unloaded data matched an earlier requested-but-absent observation")
	}
	object.RequestedProperties = []string{"description"}
	guid, _ := shared.ParseGUID("ec93656c-f745-4926-9bbb-31e7bbecb68b")
	object.Attributes["objectguid"] = [][]byte{guid.Bytes()}
	if acknowledgementMatches(acks, change, object) {
		t.Fatal("another object's omitted attribute reused an acknowledgement")
	}
}

// TestRelinquishedClearDoesNotAuthorizeReacquisition requires a new explicit clear after ownership was removed.
func TestRelinquishedClearDoesNotAuthorizeReacquisition(t *testing.T) {
	t.Parallel()
	directory := newUnitDirectory()
	directory.name, directory.parent, directory.class = "fixture", "OU=first,DC=unit,DC=invalid", "contact"
	object := directory.snapshot([]string{"description"})
	change := shared.AttributeChange{Attribute: "description", Kind: shared.AttributeClear}
	hashed := hashedChange(change)
	stored := acknowledgements{changeKey(hashed): {GUID: fixtureGUID, IntentHash: intentHash(hashed), ObservationHash: observationHash(object, "description")}}
	model := emptyModel()
	model.ID, model.Name, model.ParentDN, model.ObjectClass = types.StringValue(fixtureGUID), types.StringValue("fixture"),
		types.StringValue(directory.parent), types.StringValue("contact")
	active := eligibleAcknowledgements(&model, stored)
	changes, _, diagnostics := effectiveChanges([]shared.AttributeChange{change}, object, active)
	if diagnostics.HasError() || len(changes) != 1 {
		t.Fatal("a stale acknowledgement silently authorized re-acquired clear intent from omission")
	}
	model.ClearAttributes = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("description")})
	active = eligibleAcknowledgements(&model, stored)
	changes, _, diagnostics = effectiveChanges([]shared.AttributeChange{change}, object, active)
	if diagnostics.HasError() || len(changes) != 0 {
		t.Fatal("continuously owned unchanged accepted clear intent was replayed")
	}
}
