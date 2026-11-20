// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account/mocks"
)

var errPrivateMutation = errors.New("private directory error")

type memoryPending struct {
	data []byte
	fail bool
}

func (m *memoryPending) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return m.data, nil
}

func (m *memoryPending) SetKey(_ context.Context, _ string, value []byte) diag.Diagnostics {
	if m.fail {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Private storage rejected", "Synthetic persistence refusal.")}
	}
	m.data = append([]byte(nil), value...)
	return nil
}

// TestLastAssociationAcknowledgementDoesNotRequireReadableEmptyAttribute mirrors the SDK's accepted lastHidden outcome without inventing empty visibility.
func TestLastAssociationAcknowledgementDoesNotRequireReadableEmptyAttribute(t *testing.T) {
	for _, mode := range []string{"accepted-omitted", "mixed-failure", "uncertain", "partial", "contradictory-version"} {
		t.Run(mode, func(t *testing.T) {
			directory := unitDirectory(t)
			directory.SetLink(computerID, accountID, true)
			storage := &memoryPending{}
			service := &mocks.Service{
				Resolve: directory.ResolveIdentity, Assert: directory.Search, Links: directory.ComputerServiceAccounts,
				Remove: func(ctx context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
					result, _ := directory.RemoveComputerServiceAccounts(ctx, request)
					guid, _ := shared.ParseGUID(computerID)
					result.Links = &sdkaccounts.ComputerServiceAccountsResult{
						Version: &shared.ObjectVersion{GUID: guid}, Complete: true,
						Ranges: []sdkaccounts.ComputerServiceAccountRange{{Property: "msDS-HostServiceAccount;range=0-1499"}},
					}
					switch mode {
					case "mixed-failure":
						return result, errors.Join(shared.ErrObjectVerification, errPrivateMutation)
					case "uncertain":
						result.Mutation.State = shared.ObjectWriteUncertain
					case "partial":
						result.Mutation.PartialEffectsPossible = true
					case "contradictory-version":
						other, _ := shared.ParseGUID(otherID)
						result.Links.Version.GUID = other
					}
					return result, shared.ErrObjectVerification
				},
			}
			r := &ComputerServiceAccountResource{client: service}
			model := ComputerServiceAccountResourceModel{ComputerGUID: types.StringValue(computerID), AccountGUID: types.StringValue(accountID), ID: types.StringNull()}
			computer, account, id, _ := constructPair(model)
			retained, diagnostics := r.change(t.Context(), &model, storage, computer, account, id, false)
			if !retained || directory.Removes != 1 || directory.HasLink(computerID, accountID) {
				t.Fatal("last-link submission was replayed or lost its exact pair")
			}
			if mode == "accepted-omitted" {
				if diagnostics.HasError() || len(storage.data) != 0 || model.IsAssociated.IsNull() || model.IsAssociated.ValueBool() {
					t.Fatal("an acknowledged value deletion was blocked by an unreadable empty attribute", diagnostics)
				}
			} else if !diagnostics.HasError() || len(storage.data) == 0 {
				t.Fatal("failed, uncertain or conflicting evidence was mistaken for acknowledged deletion", diagnostics)
			}
		})
	}
}

// TestUncertainAdditionReconcilesWithoutReplay retains intent after failed readback and resolves only through fresh same-pair proof.
func TestUncertainAdditionReconcilesWithoutReplay(t *testing.T) {
	for _, immediatelyVisible := range []bool{false, true} {
		directory := unitDirectory(t)
		storage := &memoryPending{}
		written := false
		service := &mocks.Service{Resolve: directory.ResolveIdentity, Assert: func(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
			if written && !immediatelyVisible {
				return shared.SearchResult{}, shared.ErrIncompleteResult
			}
			return directory.Search(ctx, options)
		}, Add: func(ctx context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
			result, _ := directory.AddComputerServiceAccounts(ctx, request)
			written = true
			result.Mutation.State = shared.ObjectWriteUncertain
			return result, shared.ErrObjectWriteUncertain
		}}
		r := &ComputerServiceAccountResource{client: service}
		model := ComputerServiceAccountResourceModel{
			ID:           types.StringValue(computerID + "/" + accountID),
			ComputerGUID: types.StringValue(computerID),
			AccountGUID:  types.StringValue(accountID),
		}
		computer, account, id, diagnostics := constructPair(model)
		if diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		retained, diagnostics := r.change(t.Context(), &model, storage, computer, account, id, true)
		if !retained || directory.Adds != 1 || diagnostics.HasError() == immediatelyVisible {
			t.Fatal("uncertain mutation effects or readback status were misreported", diagnostics)
		}
		if immediatelyVisible {
			if len(storage.data) != 0 {
				t.Fatal("positively reconciled intent was not cleared")
			}
			continue
		}
		if len(storage.data) == 0 || !reconcilePending(t.Context(), storage, id, common.Edge{}).HasError() {
			t.Fatal("failed observation permitted another write")
		}
		edge, err := common.Read(t.Context(), directory, computer, account)
		if err != nil || reconcilePending(t.Context(), storage, id, edge).HasError() || len(storage.data) != 0 || directory.Adds != 1 {
			t.Fatal("fresh same-pair proof did not reconcile without replay", err)
		}
	}
}

// TestContradictoryMutationIdentityStaysPending preserves evidence of possible effects on another computer despite a later matching intended link.
func TestContradictoryMutationIdentityStaysPending(t *testing.T) {
	directory := unitDirectory(t)
	storage := &memoryPending{}
	other, _ := shared.ParseGUID(otherID)
	service := &mocks.Service{
		Resolve: directory.ResolveIdentity,
		Assert:  directory.Search,
		Add: func(ctx context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
			result, err := directory.AddComputerServiceAccounts(ctx, request)
			result.Mutation.TargetGUID = &other
			if err != nil {
				return result, fmt.Errorf("add synthetic account link: %w", err)
			}
			return result, nil
		},
	}
	r := &ComputerServiceAccountResource{client: service}
	model := ComputerServiceAccountResourceModel{ComputerGUID: types.StringValue(computerID), AccountGUID: types.StringValue(accountID), ID: types.StringNull()}
	computer, account, id, _ := constructPair(model)
	retained, diagnostics := r.change(t.Context(), &model, storage, computer, account, id, true)
	if !retained || !diagnostics.HasError() {
		t.Fatal("contradictory positive mutation identity was discarded")
	}
	edge, err := common.Read(t.Context(), directory, computer, account)
	if err != nil || !reconcilePending(t.Context(), storage, id, edge).HasError() || len(storage.data) == 0 || directory.Adds != 1 {
		t.Fatal("later intended-link proof erased another-computer evidence")
	}
}

// TestMissingDurableStorageAndRejectedWritesPreserveSubmissionBoundary refuses uncheckpointed writes and clears known rejections.
func TestMissingDurableStorageAndRejectedWritesPreserveSubmissionBoundary(t *testing.T) {
	directory := unitDirectory(t)
	r := &ComputerServiceAccountResource{client: directory}
	model := ComputerServiceAccountResourceModel{ComputerGUID: types.StringValue(computerID), AccountGUID: types.StringValue(accountID), ID: types.StringNull()}
	computer, account, id, _ := constructPair(model)
	for _, storage := range []pendingData{nil, &memoryPending{fail: true}} {
		if retained, diagnostics := r.change(t.Context(), &model, storage, computer, account, id, true); retained || !diagnostics.HasError() || directory.Adds != 0 {
			t.Fatal("missing durable intent permitted a write")
		}
	}
	storage := &memoryPending{}
	r.client = &mocks.Service{Add: func(context.Context, sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
		return sdkaccounts.ComputerServiceAccountMutationResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteRejected}}, errPrivateMutation
	}}
	if retained, diagnostics := r.change(t.Context(), &model, storage, computer, account, id, true); retained || !diagnostics.HasError() || len(storage.data) != 0 {
		t.Fatal("known rejection became accepted or left ambiguous intent")
	}
}
