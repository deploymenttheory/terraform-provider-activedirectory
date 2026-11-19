// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

const fixtureGUID = "a3e9767a-200e-43dd-bb1d-e5ab400e5d6f"

var errPrivateFixture = errors.New("private-fixture-password")

func TestCreateRetainsAcceptedIdentityAndDoesNotReplayAfterProtectionFailure(t *testing.T) {
	guid, err := shared.ParseGUID(fixtureGUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, knownGUID := range []bool{true, false} {
		model := minimalModel()
		calls := 0
		r := &OrganizationalUnitResource{
			client: &featuremocks.Service{
				Create: func(_ context.Context, request organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error) {
					calls++
					if request.Options.ExpectedVersion != nil || request.Parent.Form() != shared.IdentityFormDN {
						t.Fatal("ordinary Create changed SDK preconditions or guessed parent identity")
					}
					result := organizational_units.OrganizationalUnitMutationResult{Object: shared.ObjectMutationResult{
						State: shared.ObjectWriteAccepted, TargetDN: "OU=fixture,DC=unit,DC=invalid",
					}}
					if knownGUID {
						result.Object.TargetGUID = &guid
					}
					return result, errors.Join(organizational_units.ErrOUProtection, errPrivateFixture)
				},
			},
		}
		request, response := createRequest(t, r, model)
		r.Create(t.Context(), request, &response)
		var state OrganizationalUnitResourceModel
		if diagnostics := response.State.Get(t.Context(), &state); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		if calls != 1 || !response.Diagnostics.HasError() {
			t.Fatal("accepted partial creation was replayed or recorded as complete")
		}
		if knownGUID && state.ID.ValueString() != fixtureGUID {
			t.Fatal("accepted GUID was lost after protection failure")
		}
		if !knownGUID && (state.RecoveryDN.ValueString() != "OU=fixture,DC=unit,DC=invalid" || !state.ID.IsNull()) {
			t.Fatal("GUID-less accepted creation lost recovery state")
		}
	}
}

func TestUnknownPlanPreventsEveryCreateCall(t *testing.T) {
	model := minimalModel()
	model.Description = types.StringUnknown()
	calls := 0
	r := &OrganizationalUnitResource{
		client: &featuremocks.Service{
			Create: func(context.Context, organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error) {
				calls++
				return organizational_units.OrganizationalUnitMutationResult{}, nil
			},
		},
	}
	request, response := createRequest(t, r, model)
	r.Create(t.Context(), request, &response)
	if calls != 0 || !response.Diagnostics.HasError() {
		t.Fatal("unknown plan reached SDK mutation")
	}
}

func TestHiddenAttributesAndEscapedDNProjection(t *testing.T) {
	guid, err := shared.ParseGUID(fixtureGUID)
	if err != nil {
		t.Fatal(err)
	}
	model := minimalModel()
	model.ID = types.StringValue(fixtureGUID)
	model.Description = types.StringValue("retained")
	name, protect := "fixture", false
	unit := &organizational_units.OrganizationalUnit{
		GUID: &guid, Name: &name, ProtectedFromAccidentalDeletion: &protect,
		DistinguishedName: `OU=escaped\,name,OU=parent,DC=unit,DC=invalid`,
		Raw:               shared.Object{RequestedProperties: []string{"description"}},
	}
	diagnostics := mapState(&model, unit)
	if diagnostics.HasError() || model.Description.ValueString() != "retained" || model.ParentDN.ValueString() != "OU=parent,DC=unit,DC=invalid" {
		t.Fatalf("hidden property/location observation was lost: %v", diagnostics)
	}
}

func TestReadDenialAndLeafDeleteFailureRetainState(t *testing.T) {
	model := minimalModel()
	model.ID = types.StringValue(fixtureGUID)
	model.DistinguishedName = types.StringValue("OU=fixture,DC=unit,DC=invalid")
	r := &OrganizationalUnitResource{client: &featuremocks.Service{
		Read: func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
			return organizational_units.OrganizationalUnitResult{}, errors.Join(shared.ErrDirectoryObjectNotFound, shared.ErrObjectWritePermission)
		},
		Delete: func(_ context.Context, request organizational_units.DeleteOrganizationalUnitRequest) (shared.ObjectMutationResult, error) {
			if request.Recursive || request.Options.ExpectedVersion != nil {
				t.Fatal("leaf delete silently became recursive or gained a version assertion")
			}
			return shared.ObjectMutationResult{State: shared.ObjectWriteRejected}, shared.ErrObjectNotLeaf
		},
	}}
	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diagnostics := state.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	read := resource.ReadResponse{State: state}
	r.Read(t.Context(), resource.ReadRequest{State: state}, &read)
	if !read.Diagnostics.HasError() || read.State.Raw.IsNull() {
		t.Fatal("denied read removed state")
	}
	deleted := resource.DeleteResponse{State: state}
	r.Delete(t.Context(), resource.DeleteRequest{State: state}, &deleted)
	if !deleted.Diagnostics.HasError() || deleted.State.Raw.IsNull() {
		t.Fatal("nonempty OU was treated as deleted")
	}
}

func TestAcceptedRenameSurvivesLaterDeniedMove(t *testing.T) {
	guid, err := shared.ParseGUID(fixtureGUID)
	if err != nil {
		t.Fatal(err)
	}
	current := minimalModel()
	current.ID = types.StringValue(fixtureGUID)
	current.ParentDN = types.StringValue("OU=first,DC=unit,DC=invalid")
	current.DistinguishedName = types.StringValue("OU=fixture,OU=first,DC=unit,DC=invalid")
	plan := current
	plan.Name = types.StringValue("renamed")
	plan.ParentDN = types.StringValue("OU=second,DC=unit,DC=invalid")
	renames, moves := 0, 0
	r := &OrganizationalUnitResource{client: &featuremocks.Service{
		Rename: func(_ context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			renames++
			if request.Identity.Form() != shared.IdentityFormGUID || request.Options.ExpectedVersion != nil {
				t.Fatal("rename lost GUID identity or added a version assertion")
			}
			return shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, Verified: true, TargetGUID: &guid}, nil
		},
		Read: func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
			name, protect := "renamed", false
			return organizational_units.OrganizationalUnitResult{
				Unit: &organizational_units.OrganizationalUnit{
					GUID: &guid, Name: &name, ProtectedFromAccidentalDeletion: &protect,
					DistinguishedName: "OU=renamed,OU=first,DC=unit,DC=invalid",
				},
				Observation: shared.IdentityResult{Complete: true},
			}, nil
		},
		Move: func(_ context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			moves++
			if request.Destination != nil || request.Options.ExpectedVersion != nil {
				t.Fatal("local move enabled cross-domain delegation or version controls")
			}
			return shared.ObjectMoveResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteRejected, TargetGUID: &guid}}, shared.ErrObjectWritePermission
		},
	}}
	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	state, planned := tfsdk.State{Schema: schemaResponse.Schema}, tfsdk.Plan{Schema: schemaResponse.Schema}
	if diagnostics := state.Set(t.Context(), &current); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if diagnostics := planned.Set(t.Context(), &plan); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(t.Context(), resource.UpdateRequest{State: state, Plan: planned}, &response)
	var after OrganizationalUnitResourceModel
	if diagnostics := response.State.Get(t.Context(), &after); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if !response.Diagnostics.HasError() || renames != 1 || moves != 1 || after.ID.ValueString() != fixtureGUID ||
		after.Name.ValueString() != "renamed" || after.ParentDN.ValueString() != current.ParentDN.ValueString() {
		t.Fatal("a later move failure discarded/replayed the accepted rename or recorded an unapplied parent")
	}
}

func minimalModel() OrganizationalUnitResourceModel {
	return OrganizationalUnitResourceModel{
		Name: types.StringValue("fixture"), ParentDN: types.StringValue("DC=unit,DC=invalid"),
		ID: types.StringUnknown(), DistinguishedName: types.StringUnknown(), RecoveryDN: types.StringUnknown(),
		ProtectedFromAccidentalDeletion: types.BoolValue(false),
	}
}

func createRequest(t *testing.T, r *OrganizationalUnitResource, model OrganizationalUnitResourceModel) (resource.CreateRequest, resource.CreateResponse) {
	t.Helper()
	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diagnostics := plan.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return resource.CreateRequest{Plan: plan}, resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema, Raw: plan.Raw}}
}
