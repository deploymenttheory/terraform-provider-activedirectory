// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkmemberships "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
)

var errPrivatePeer = errors.New("private directory credential peer text")

const (
	groupID  = "11111111-1111-4111-8111-111111111111"
	memberID = "22222222-2222-4222-8222-222222222222"
	otherID  = "33333333-3333-4333-8333-333333333333"
)

func directory(t *testing.T) *mocks.Directory {
	t.Helper()
	data, err := os.ReadFile("tests/responses/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := mocks.DecodeDirectory(data)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

// TestDirectAssertionsDistinguishAbsenceFromEmptySearch verifies only a matching server-returned group can prove either link state.
func TestDirectAssertionsDistinguishAbsenceFromEmptySearch(t *testing.T) {
	for _, present := range []bool{false, true} {
		d := directory(t)
		d.SetLink(groupID, memberID, present)
		group, _ := shared.IdentityByGUID(groupID)
		member, _ := shared.IdentityByGUID(memberID)
		edge, err := common.Read(t.Context(), d, group, member)
		if err != nil || !edge.Confirmed || edge.Present != present {
			t.Fatal("explicit direct-link assertion did not establish the requested pair", err)
		}
		for _, mode := range []string{"empty", "denied", "partial", "wrong-guid"} {
			service := &mocks.Service{Resolve: d.ResolveIdentity, Assert: func(context.Context, shared.SearchOptions) (shared.SearchResult, error) {
				switch mode {
				case "empty":
					return shared.SearchResult{Complete: true}, nil
				case "denied":
					return shared.SearchResult{}, errPrivatePeer
				case "partial":
					return shared.SearchResult{Objects: []shared.Object{d.Objects[groupID]}}, nil
				default:
					return shared.SearchResult{Objects: []shared.Object{d.Objects[memberID]}, Complete: true}, nil
				}
			}}
			failed, err := common.Read(t.Context(), service, group, member)
			if err == nil || failed.Confirmed {
				t.Fatal("incomplete or contradictory assertion became absence proof", mode)
			}
		}
		if d.Adds != 0 || d.Removes != 0 {
			t.Fatal("read submitted a membership mutation")
		}
	}
}

// TestEndpointLookupMustRemainStable refuses positive membership proof after the member's name changes between observations.
func TestEndpointLookupMustRemainStable(t *testing.T) {
	d := directory(t)
	d.SetLink(groupID, memberID, true)
	reads := 0
	service := &mocks.Service{Assert: d.Search, Resolve: func(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
		if target.String() == memberID {
			reads++
			if reads == 2 {
				object := d.Objects[memberID]
				object.Attributes = map[string][][]byte{
					"objectguid": object.Attributes["objectguid"], "objectclass": object.Attributes["objectclass"],
					"distinguishedname": {[]byte("CN=moved,DC=unit,DC=invalid")},
				}
				d.Objects[memberID] = object
			}
		}
		return d.ResolveIdentity(ctx, target, options)
	}}
	group, _ := shared.IdentityByGUID(groupID)
	member, _ := shared.IdentityByGUID(memberID)
	result, err := common.Read(t.Context(), service, group, member)
	if err == nil || result.Confirmed {
		t.Fatal("a stale member name became stable relationship proof")
	}
}

// TestDiscoveryNeedsPositiveEmptyProof preserves unknown collections and rejects unresolved or duplicate member identities.
func TestDiscoveryNeedsPositiveEmptyProof(t *testing.T) {
	d := directory(t)
	group, _ := shared.IdentityByGUID(groupID)
	empty, err := common.Discover(t.Context(), d, group, 10)
	if err != nil || !empty.Complete || len(empty.Edges) != 0 {
		t.Fatal("positive server empty-group assertion was not accepted", err)
	}
	service := &mocks.Service{Links: d.GroupMemberLinks, Resolve: d.ResolveIdentity, Assert: func(context.Context, shared.SearchOptions) (shared.SearchResult, error) {
		return shared.SearchResult{Complete: true}, nil
	}}
	unknown, err := common.Discover(t.Context(), service, group, 10)
	if err == nil || unknown.Complete {
		t.Fatal("omitted member attribute and zero search rows became an empty group")
	}
	d.SetLink(groupID, otherID, true)
	d.SetLink(groupID, memberID, true)
	all, err := common.Discover(t.Context(), d, group, 10)
	if err != nil || !all.Complete || len(all.Edges) != 2 || all.Edges[0].Member.GUID.String() != memberID {
		t.Fatal("direct links were not independently resolved and sorted", err)
	}
	bounded, err := common.Discover(t.Context(), d, group, 1)
	if err == nil || bounded.Complete || !bounded.Bounded || len(bounded.Edges) != 1 {
		t.Fatal("bounded direct membership discovery claimed completion", err)
	}
}

// TestInvalidPairsAndBoundsRejectUnknownValues keeps malformed identities and unbounded discovery out of the SDK.
func TestInvalidPairsAndBoundsRejectUnknownValues(t *testing.T) {
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("CN=name,DC=unit"), types.StringValue("00000000-0000-0000-0000-000000000000")} {
		_, _, diagnostics := common.ConstructPair(types.StringValue(groupID), value)
		if !diagnostics.HasError() {
			t.Fatal("invalid member GUID was accepted", value)
		}
	}
	for _, limit := range []types.Int64{types.Int64Unknown(), types.Int64Value(0), types.Int64Value(-1), types.Int64Value(10001)} {
		_, _, diagnostics := common.ConstructQuery(common.QueryModel{GroupGUID: types.StringValue(groupID), ResultLimit: limit})
		if !diagnostics.HasError() {
			t.Fatal("unknown or unbounded membership discovery was accepted", limit)
		}
	}
}

// TestDiscoveryUsesExplicitGroupMetadata exercises the SDK range contract, which does not select a distinguishedName attribute.
func TestDiscoveryUsesExplicitGroupMetadata(t *testing.T) {
	d := directory(t)
	d.SetLink(groupID, memberID, true)
	service := &mocks.Service{
		Resolve: d.ResolveIdentity,
		Assert:  d.Search,
		Links: func(ctx context.Context, target shared.Identity, options sdkmemberships.GroupMemberLinksOptions) (sdkmemberships.GroupMemberLinksResult, error) {
			result, err := d.GroupMemberLinks(ctx, target, options)
			if err != nil {
				return result, fmt.Errorf("read synthetic SDK range: %w", err)
			}
			original := *result.Observation.Object
			original.Attributes = map[string][][]byte{
				"objectguid": original.Attributes["objectguid"], "objectclass": original.Attributes["objectclass"],
			}
			result.Observation.Object = &original
			result.Observation.Candidates = []shared.Object{original}
			return result, nil
		},
	}
	group, _ := shared.IdentityByGUID(groupID)
	observed, err := common.Discover(t.Context(), service, group, 10)
	if err != nil || !observed.Complete || len(observed.Edges) != 1 || observed.Group.DN != "CN=group,DC=unit,DC=invalid" {
		t.Fatal("SDK range metadata was treated as a guarantee of explicitly selected group properties", err)
	}
}
