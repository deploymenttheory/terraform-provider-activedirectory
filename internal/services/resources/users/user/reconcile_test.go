// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type memoryPending struct {
	data   []byte
	reject bool
}

func (p *memoryPending) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return p.data, nil
}

func (p *memoryPending) SetKey(_ context.Context, _ string, data []byte) diag.Diagnostics {
	if p.reject {
		return diag.Diagnostics{diag.NewErrorDiagnostic("fixture storage unavailable", "synthetic private storage refusal")}
	}
	p.data = append([]byte(nil), data...)
	return nil
}

func TestReconciliationRefusesUnobservablePasswordCreationAndClear(t *testing.T) {
	directory := newUnitDirectory(t)
	observed := directory.snapshot(t)
	for _, operation := range []string{"create", "password", "delete", "update"} {
		p := &memoryPending{}
		pending := pendingMutation{Operation: operation, GUID: fixtureGUID, PasswordVersion: 2}
		if operation == "update" {
			pending.Properties = map[string]*string{"description": nil}
		}
		if diagnostics := savePending(t.Context(), p, pending); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		if diagnostics := reconcilePending(t.Context(), p, observed); !diagnostics.HasError() || len(p.data) == 0 {
			t.Fatal("unobservable effect was treated as verified and intent discarded")
		}
		if diagnostics := guardPending(t.Context(), p); !diagnostics.HasError() {
			t.Fatal("unresolved write was permitted to replay")
		}
	}
}

func TestReconciliationRequiresObservedIdentityAndRequestedValues(t *testing.T) {
	directory := newUnitDirectory(t)
	observed := directory.snapshot(t)
	value := "changed"
	p := &memoryPending{}
	if diagnostics := savePending(
		t.Context(),
		p,
		pendingMutation{Operation: "update", GUID: fixtureGUID, Properties: map[string]*string{"description": &value}},
	); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if diagnostics := reconcilePending(t.Context(), p, observed); !diagnostics.HasError() {
		t.Fatal("missing property proved a write")
	}
	observed.Raw.Attributes["description"] = [][]byte{[]byte(value)}
	if diagnostics := reconcilePending(t.Context(), p, observed); diagnostics.HasError() || len(p.data) != 0 {
		t.Fatal("matching complete read did not settle observable intent")
	}
}

func TestMissingPrivateStorageRefusesMutationIntent(t *testing.T) {
	if diagnostics := savePending(t.Context(), nil, pendingMutation{Operation: "create"}); !diagnostics.HasError() {
		t.Fatal("absent recovery storage authorized a write")
	}
	if diagnostics := savePending(t.Context(), &memoryPending{reject: true}, pendingMutation{Operation: "password"}); !diagnostics.HasError() {
		t.Fatal("failed recovery storage authorized a write")
	}
}
