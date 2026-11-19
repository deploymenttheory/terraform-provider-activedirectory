// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type memoryPending struct {
	data []byte
}

func (p *memoryPending) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return p.data, nil
}

func (p *memoryPending) SetKey(_ context.Context, _ string, data []byte) diag.Diagnostics {
	p.data = append([]byte(nil), data...)
	return nil
}

func TestGroupPendingIntentRequiresStorageAndReadProof(t *testing.T) {
	if diagnostics := savePending(t.Context(), nil, pendingMutation{Operation: "create"}); !diagnostics.HasError() {
		t.Fatal("missing recovery storage allowed mutation intent to disappear")
	}
	directory := newUnitDirectory(t)
	directory.flags |= 0x80000002
	observed := directory.snapshot()
	for _, operation := range []string{"create", "delete", "update"} {
		storage := &memoryPending{}
		pending := pendingMutation{Operation: operation, GUID: fixtureGUID}
		if operation == "update" {
			pending.Properties = map[string]*string{"description": nil}
		}
		if diagnostics := savePending(t.Context(), storage, pending); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		if diagnostics := reconcilePending(t.Context(), storage, observed); !diagnostics.HasError() || len(storage.data) == 0 {
			t.Fatal("failed creation, pending deletion or uncertain clear was settled without proof")
		}
		if diagnostics := guardPending(t.Context(), storage); !diagnostics.HasError() {
			t.Fatal("unresolved intent authorized a write replay")
		}
	}
	storage := &memoryPending{}
	scope := "universal"
	if diagnostics := savePending(t.Context(), storage, pendingMutation{Operation: "update", GUID: fixtureGUID, Scope: &scope}); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if diagnostics := reconcilePending(t.Context(), storage, observed); !diagnostics.HasError() {
		t.Fatal("an observed global group proved a pending universal conversion")
	}
	known := groups.GroupUniversal
	observed.Scope = &known
	if diagnostics := reconcilePending(t.Context(), storage, observed); diagnostics.HasError() || len(storage.data) != 0 {
		t.Fatal("complete same-GUID scope proof did not settle its checkpoint")
	}
}

func TestGroupNoopCannotAdoptAnotherReadbackGUID(t *testing.T) {
	model := creationModel()
	model.ID = types.StringValue(fixtureGUID)
	directory := newUnitDirectory(t)
	result := directory.mutation()
	result.State = shared.ObjectWriteNotAttempted
	other, _ := shared.ParseGUID("dc93656c-f745-4926-9bbb-31e7bbecb68b")
	result.After.Attributes["objectguid"] = [][]byte{other.Bytes()}
	if diagnostics := recordMutation(&model, result, nil); !diagnostics.HasError() || model.ID.ValueString() != fixtureGUID {
		t.Fatal("verified no-op adopted a different object identity")
	}
}
