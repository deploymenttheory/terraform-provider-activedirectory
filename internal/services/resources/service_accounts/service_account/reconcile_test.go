// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type memoryPending struct{ data []byte }

func (p *memoryPending) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return p.data, nil
}

func (p *memoryPending) SetKey(_ context.Context, _ string, data []byte) diag.Diagnostics {
	p.data = append([]byte(nil), data...)
	return nil
}

// TestReaderRecoveryRequiresReturnedDescriptor distinguishes hidden policy from a complete empty policy after uncertain submission.
func TestReaderRecoveryRequiresReturnedDescriptor(t *testing.T) {
	directory := newUnitDirectory()
	observed := directory.snapshot(t)
	empty := []string{}
	storage := &memoryPending{}
	pending := pendingMutation{Operation: "update", GUID: fixtureGUID, Readers: &empty}
	if diagnostics := savePending(t.Context(), storage, pending); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	observed.PasswordReaderDescriptorReturned = false
	if diagnostics := reconcilePending(t.Context(), storage, observed); !diagnostics.HasError() || len(storage.data) == 0 {
		t.Fatal("hidden descriptor proved an empty reader policy")
	}
	observed.PasswordReaderDescriptorReturned = true
	if diagnostics := reconcilePending(t.Context(), storage, observed); diagnostics.HasError() || len(storage.data) != 0 {
		t.Fatal("positively returned empty descriptor did not settle reader intent")
	}
}

// TestRecoveryDoesNotReplayCreationOrInferClears protects accepted identity and unresolved effects.
func TestRecoveryDoesNotReplayCreationOrInferClears(t *testing.T) {
	observed := newUnitDirectory().snapshot(t)
	for _, pending := range []pendingMutation{
		{Operation: "create", GUID: fixtureGUID},
		{Operation: "update", GUID: fixtureGUID, Properties: map[string]*string{"description": nil}},
		{Operation: "update", GUID: fixtureGUID, SPNs: &[]string{}},
	} {
		storage := &memoryPending{}
		if diagnostics := savePending(t.Context(), storage, pending); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		if diagnostics := reconcilePending(t.Context(), storage, observed); !diagnostics.HasError() {
			t.Fatal("unobservable creation or clear was treated as settled")
		}
		if diagnostics := guardPending(t.Context(), storage); !diagnostics.HasError() {
			t.Fatal("unresolved intent authorized a replay or tainted replacement")
		}
	}
	if diagnostics := savePending(t.Context(), nil, pendingMutation{Operation: "create"}); !diagnostics.HasError() {
		t.Fatal("missing recovery storage authorized a write")
	}
}

// TestEncryptionRecoveryIgnoresUnownedBits verifies a selected low-bit policy independently of compound and unknown flags.
func TestEncryptionRecoveryIgnoresUnownedBits(t *testing.T) {
	observed := newUnitDirectory().snapshot(t)
	actual, want := shared.Flags32(0x20018), uint32(0x18)
	observed.SupportedEncryptionTypes = &actual
	if !matchesPendingProperties(&pendingMutation{Encryption: &want}, observed) {
		t.Fatal("unrelated flags prevented reconciliation of the selected policy bits")
	}
	actual = 0x20010
	if matchesPendingProperties(&pendingMutation{Encryption: &want}, observed) {
		t.Fatal("a different encryption policy settled the checkpoint")
	}
}

// TestFailedStandaloneEnableRetainsPasswordAcknowledgement proves later errors cannot lose or replace an accepted creation GUID.
func TestFailedStandaloneEnableRetainsPasswordAcknowledgement(t *testing.T) {
	model := creationModel()
	model.ID = types.StringValue(fixtureGUID)
	model.Kind = types.StringValue("standalone")
	guid, _ := shared.ParseGUID(fixtureGUID)
	result := service_accounts.ServiceAccountCreationResult{Password: &accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid}}
	diagnostics := observeCreationStages(&model, 7, service_accounts.ServiceAccountStandalone, true, result, shared.ErrObjectVerification)
	if !diagnostics.HasError() || model.ID.ValueString() != fixtureGUID || model.PasswordWOVersion.ValueInt64() != 7 {
		t.Fatal("failed enablement lost accepted identity or password-stage acknowledgement")
	}
	other, _ := shared.ParseGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	result.Password.TargetGUID = &other
	if diagnostics := observeCreationStages(&model, 7, service_accounts.ServiceAccountStandalone, false, result, nil); !diagnostics.HasError() ||
		model.PasswordWOVersion.ValueInt64() != 0 {
		t.Fatal("a different GUID's password acknowledgement was adopted")
	}
}
