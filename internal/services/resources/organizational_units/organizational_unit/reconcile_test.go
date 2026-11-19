// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

type pendingFixture map[string][]byte

func (p pendingFixture) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return p[key], nil
}

func (p pendingFixture) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	p[key] = value
	return nil
}

func TestPrivateIntentBlocksReplayUntilObservedEffectsMatch(t *testing.T) {
	state := minimalModel()
	state.ID = types.StringValue(fixtureGUID)
	storage := pendingFixture{}
	value := "desired"
	pending := pendingMutation{Operation: "update", GUID: fixtureGUID, Properties: map[string]*string{"description": &value}}
	if diagnostics := savePending(t.Context(), storage, pending); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if !guardPending(t.Context(), storage).HasError() {
		t.Fatal("unsettled write did not block another mutation")
	}
	if diagnostics := finishPending(
		t.Context(),
		storage,
		pending,
		&state,
		shared.ObjectMutationResult{State: shared.ObjectWriteUncertain},
		shared.ErrObjectWriteUncertain,
		false,
		false,
	); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	guid, err := shared.ParseGUID(fixtureGUID)
	if err != nil {
		t.Fatal(err)
	}
	name := "fixture"
	unit := &organizational_units.OrganizationalUnit{
		GUID: &guid, Name: &name,
		Raw: shared.Object{Attributes: map[string][][]byte{"description": {[]byte("old")}}},
	}
	if !reconcilePending(t.Context(), storage, unit).HasError() || len(storage[pendingKey]) == 0 {
		t.Fatal("contradictory read silently authorized write replay")
	}
	unit.Raw.Attributes["description"] = [][]byte{[]byte(value)}
	if diagnostics := reconcilePending(t.Context(), storage, unit); diagnostics.HasError() || len(storage[pendingKey]) != 0 {
		t.Fatal("verified current effect did not settle pending intent")
	}
	if guardPending(t.Context(), storage).HasError() {
		t.Fatal("settled intent still blocked writes")
	}
	for _, unresolved := range []pendingMutation{
		{Operation: "update", GUID: fixtureGUID, Properties: map[string]*string{"description": nil}},
		{Operation: "create", Name: "fixture"},
		{Operation: "delete", GUID: fixtureGUID},
	} {
		if diagnostics := savePending(t.Context(), storage, unresolved); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		unit.Raw = shared.Object{RequestedProperties: []string{"description"}}
		if !reconcilePending(t.Context(), storage, unit).HasError() {
			t.Fatal("absence, DN candidate or live delete target was treated as proof of the pending operation")
		}
	}
	if diagnostics := finishPending(
		t.Context(),
		storage,
		pending,
		&state,
		shared.ObjectMutationResult{State: shared.ObjectWriteRejected},
		shared.ErrObjectWritePermission,
		false,
		false,
	); diagnostics.HasError() ||
		len(storage[pendingKey]) != 0 {
		t.Fatal("known rejection without effects was recorded as uncertain")
	}
	if diagnostics := finishPending(
		t.Context(),
		storage,
		pending,
		&state,
		shared.ObjectMutationResult{State: shared.ObjectWriteRejected, PartialEffectsPossible: true},
		shared.ErrObjectDeletionIncomplete,
		false,
		false,
	); diagnostics.HasError() ||
		len(storage[pendingKey]) == 0 {
		t.Fatal("partial effects after rejection were discarded")
	}
}

func TestReadDoesNotAdoptAnUnverifiedDNOnlyCreation(t *testing.T) {
	state := minimalModel()
	state.ID = types.StringNull()
	state.DistinguishedName = types.StringNull()
	state.RecoveryDN = types.StringValue("OU=fixture,DC=unit,DC=invalid")
	reads := 0
	r := &OrganizationalUnitResource{
		client: &mocks.Service{Read: func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
			reads++
			return organizational_units.OrganizationalUnitResult{}, nil
		}},
	}
	var schema resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schema)
	prior := tfsdk.State{Schema: schema.Schema}
	if diagnostics := prior.Set(t.Context(), &state); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	response := resource.ReadResponse{State: prior}
	r.Read(t.Context(), resource.ReadRequest{State: prior}, &response)
	if reads != 0 || !response.Diagnostics.HasError() || response.State.Raw.IsNull() {
		t.Fatal("DN-only recovery adopted an unverified candidate or discarded recovery state")
	}
}
