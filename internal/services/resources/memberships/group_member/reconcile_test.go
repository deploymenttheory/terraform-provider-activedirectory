// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
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
		}, Add: func(ctx context.Context, request memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
			result, _ := directory.AddGroupMembers(ctx, request)
			written = true
			result.Mutation.State = shared.ObjectWriteUncertain
			return result, shared.ErrObjectWriteUncertain
		}}
		r := &GroupMemberResource{client: service}
		model := GroupMemberResourceModel{ID: types.StringValue(groupID + "/" + memberID), GroupGUID: types.StringValue(groupID), MemberGUID: types.StringValue(memberID)}
		group, member, id, diagnostics := constructPair(model)
		if diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		retained, diagnostics := r.change(t.Context(), &model, storage, group, member, id, true)
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
		edge, err := common.Read(t.Context(), directory, group, member)
		if err != nil || reconcilePending(t.Context(), storage, id, edge).HasError() || len(storage.data) != 0 || directory.Adds != 1 {
			t.Fatal("fresh same-pair proof did not reconcile without replay", err)
		}
	}
}

// TestContradictoryMutationIdentityStaysPending preserves evidence of possible effects on another group despite a later matching intended link.
func TestContradictoryMutationIdentityStaysPending(t *testing.T) {
	directory := unitDirectory(t)
	storage := &memoryPending{}
	other, _ := shared.ParseGUID(otherID)
	service := &mocks.Service{
		Resolve: directory.ResolveIdentity,
		Assert:  directory.Search,
		Add: func(ctx context.Context, request memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
			result, err := directory.AddGroupMembers(ctx, request)
			result.Mutation.TargetGUID = &other
			if err != nil {
				return result, fmt.Errorf("add synthetic member link: %w", err)
			}
			return result, nil
		},
	}
	r := &GroupMemberResource{client: service}
	model := GroupMemberResourceModel{GroupGUID: types.StringValue(groupID), MemberGUID: types.StringValue(memberID), ID: types.StringNull()}
	group, member, id, _ := constructPair(model)
	retained, diagnostics := r.change(t.Context(), &model, storage, group, member, id, true)
	if !retained || !diagnostics.HasError() {
		t.Fatal("contradictory positive mutation identity was discarded")
	}
	edge, err := common.Read(t.Context(), directory, group, member)
	if err != nil || !reconcilePending(t.Context(), storage, id, edge).HasError() || len(storage.data) == 0 || directory.Adds != 1 {
		t.Fatal("later intended-link proof erased another-group evidence")
	}
}

// TestMissingDurableStorageAndRejectedWritesPreserveSubmissionBoundary refuses uncheckpointed writes and clears known rejections.
func TestMissingDurableStorageAndRejectedWritesPreserveSubmissionBoundary(t *testing.T) {
	directory := unitDirectory(t)
	r := &GroupMemberResource{client: directory}
	model := GroupMemberResourceModel{GroupGUID: types.StringValue(groupID), MemberGUID: types.StringValue(memberID), ID: types.StringNull()}
	group, member, id, _ := constructPair(model)
	for _, storage := range []pendingData{nil, &memoryPending{fail: true}} {
		if retained, diagnostics := r.change(t.Context(), &model, storage, group, member, id, true); retained || !diagnostics.HasError() || directory.Adds != 0 {
			t.Fatal("missing durable intent permitted a write")
		}
	}
	storage := &memoryPending{}
	r.client = &mocks.Service{Add: func(context.Context, memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
		return memberships.GroupMembershipMutationResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteRejected}}, errPrivateMutation
	}}
	if retained, diagnostics := r.change(t.Context(), &model, storage, group, member, id, true); retained || !diagnostics.HasError() || len(storage.data) != 0 {
		t.Fatal("known rejection became accepted or left ambiguous intent")
	}
}
