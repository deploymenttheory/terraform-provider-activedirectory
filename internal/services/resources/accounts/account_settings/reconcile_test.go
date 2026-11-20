// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type memoryPending struct {
	data   []byte
	reject bool
}

func (m *memoryPending) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return m.data, nil
}

func (m *memoryPending) SetKey(_ context.Context, _ string, value []byte) diag.Diagnostics {
	if m.reject {
		return diag.Diagnostics{diag.NewErrorDiagnostic("synthetic private storage refusal", "fixture-only failure")}
	}
	m.data = append([]byte(nil), value...)
	return nil
}

func enableIntent(t *testing.T) settingsIntent {
	t.Helper()
	intent, diagnostics := constructSettings(
		&AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID), Enabled: types.BoolValue(true), ExpiresAt: types.StringValue("2031-04-05T11:34:56Z")},
	)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return intent
}

// TestAcceptedPhaseRetainsEffectsAndResumesOnlyAfterVisibleProof verifies failed readback cannot authorize a later phase or replay.
func TestAcceptedPhaseRetainsEffectsAndResumesOnlyAfterVisibleProof(t *testing.T) {
	directory := newUnitDirectory(t)
	service := directory.callbacks()
	update, read := service.UpdateControl, service.ReadControl
	failRead := false
	service.ReadControl = func(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
		if failRead {
			return accounts.AccountControlResult{}, shared.ErrObjectWritePermission
		}
		return read(ctx, target, options)
	}
	service.UpdateControl = func(ctx context.Context, request accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
		result, err := update(ctx, request)
		failRead = true
		return result, err
	}
	resource := &AccountSettingsResource{client: service}
	model := AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID)}
	storage := &memoryPending{}
	intent := enableIntent(t)
	adopted, diagnostics := resource.applySettings(t.Context(), &model, intent, storage)
	if !adopted || !diagnostics.HasError() || model.ID.ValueString() != fixtureGUID || !model.Enabled.ValueBool() || len(storage.data) == 0 ||
		directory.calls["set_expiration"] != 0 {
		t.Fatal("accepted identity/effects were lost or a later phase ran after failed readback")
	}
	failRead = false
	directory.hidden["enabled"] = true
	_, diagnostics = resource.applySettings(t.Context(), &model, intent, storage)
	if !diagnostics.HasError() || directory.calls["flags"] != 1 || len(storage.data) == 0 {
		t.Fatal("retained prior state was used as proof of a hidden effect")
	}
	directory.hidden["enabled"] = false
	_, diagnostics = resource.applySettings(t.Context(), &model, intent, storage)
	if diagnostics.HasError() || directory.calls["flags"] != 1 || directory.calls["set_expiration"] != 1 || len(storage.data) != 0 {
		t.Fatal("fresh proof did not permit the later configured phase without replaying flags")
	}
}

// TestMissingPrivateStorageAndHiddenLaterPhasePreventWrites validates all phase prerequisites before the earlier enable change.
func TestMissingPrivateStorageAndHiddenLaterPhasePreventWrites(t *testing.T) {
	for _, mode := range []string{"nil-storage", "rejected-storage", "hidden-expiry"} {
		t.Run(mode, func(t *testing.T) {
			directory := newUnitDirectory(t)
			resource := &AccountSettingsResource{client: directory.callbacks()}
			var storage pendingData
			if mode != "nil-storage" {
				storage = &memoryPending{reject: mode == "rejected-storage"}
			}
			directory.hidden["expiration"] = mode == "hidden-expiry"
			model := AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID)}
			adopted, diagnostics := resource.applySettings(t.Context(), &model, enableIntent(t), storage)
			if !adopted || !diagnostics.HasError() || directory.calls["flags"] != 0 || directory.calls["set_expiration"] != 0 {
				t.Fatal("unavailable durable intent or a hidden later phase allowed an SDK mutation")
			}
		})
	}
}

// TestContradictoryWriteTargetIsSticky retains the reported other GUID even when the intended account later matches.
func TestContradictoryWriteTargetIsSticky(t *testing.T) {
	directory := newUnitDirectory(t)
	service := directory.callbacks()
	update := service.UpdateControl
	service.UpdateControl = func(ctx context.Context, request accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
		result, err := update(ctx, request)
		other, _ := shared.ParseGUID(secondGUID)
		result.Mutation.TargetGUID = &other
		return result, err
	}
	resource := &AccountSettingsResource{client: service}
	storage := &memoryPending{}
	model := AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID)}
	_, diagnostics := resource.applySettings(t.Context(), &model, enableIntent(t), storage)
	pending, recovery := loadPending(t.Context(), storage)
	if !diagnostics.HasError() || recovery.HasError() || pending == nil || !pending.IdentityConflict || pending.ReportedGUID != secondGUID ||
		model.ID.ValueString() != fixtureGUID {
		t.Fatal("contradictory scope was discarded or became the adopted identity")
	}
	_, diagnostics = resource.applySettings(t.Context(), &model, enableIntent(t), storage)
	if !diagnostics.HasError() || directory.calls["flags"] != 1 || directory.calls["set_expiration"] != 0 || len(storage.data) == 0 {
		t.Fatal("selected-account readback erased a contradiction about another possible write target")
	}
}

// TestInvalidRecoveryMetadataCannotAuthorizeMutation refuses future versions and read-only field intent in private state.
func TestInvalidRecoveryMetadataCannotAuthorizeMutation(t *testing.T) {
	for _, encoded := range []string{
		`{"version":2,"guid":"dc93656c-f745-4926-9bbb-31e7bbecb68b","operation":"flags","flags":{"enabled":true}}`,
		`{"version":1,"guid":"dc93656c-f745-4926-9bbb-31e7bbecb68b","operation":"flags","flags":{"locked_out":false}}`,
		`{"version":1,"guid":"dc93656c-f745-4926-9bbb-31e7bbecb68b","operation":"expiration","clearExpiration":true,"expiresAt":"2031-04-05T11:34:56Z"}`,
	} {
		directory := newUnitDirectory(t)
		resource := &AccountSettingsResource{client: directory.callbacks()}
		model := AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID)}
		storage := &memoryPending{data: []byte(encoded)}
		_, diagnostics := resource.applySettings(t.Context(), &model, enableIntent(t), storage)
		if !diagnostics.HasError() || directory.calls["flags"] != 0 || directory.calls["set_expiration"] != 0 || len(storage.data) == 0 {
			t.Fatal("invalid recovery state permitted a mutation or was discarded")
		}
	}
}
