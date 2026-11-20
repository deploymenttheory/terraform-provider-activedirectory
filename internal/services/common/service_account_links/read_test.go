// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account_links_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account/mocks"
)

var errPrivatePeer = errors.New("private directory credential peer text")

const (
	computerID = "11111111-1111-4111-8111-111111111111"
	accountID  = "22222222-2222-4222-8222-222222222222"
	otherID    = "33333333-3333-4333-8333-333333333333"
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

// TestDirectAssertionsDistinguishAbsenceFromEmptySearch verifies only a matching server-returned computer can prove either link state.
func TestDirectAssertionsDistinguishAbsenceFromEmptySearch(t *testing.T) {
	for _, present := range []bool{false, true} {
		d := directory(t)
		d.SetLink(computerID, otherID, true)
		d.SetLink(computerID, accountID, present)
		computer, _ := shared.IdentityByGUID(computerID)
		account, _ := shared.IdentityByGUID(accountID)
		edge, err := common.Read(t.Context(), d, computer, account)
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
					return shared.SearchResult{Objects: []shared.Object{d.Objects[computerID]}}, nil
				default:
					return shared.SearchResult{Objects: []shared.Object{d.Objects[accountID]}, Complete: true}, nil
				}
			}}
			failed, err := common.Read(t.Context(), service, computer, account)
			if err == nil || failed.Confirmed {
				t.Fatal("incomplete or contradictory assertion became absence proof", mode)
			}
		}
		if d.Adds != 0 || d.Removes != 0 {
			t.Fatal("read submitted a association mutation")
		}
	}
}

// TestEndpointLookupMustRemainStable refuses positive association proof after the account's name changes between observations.
func TestEndpointLookupMustRemainStable(t *testing.T) {
	d := directory(t)
	d.SetLink(computerID, accountID, true)
	reads := 0
	service := &mocks.Service{Assert: d.Search, Resolve: func(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
		if target.String() == accountID {
			reads++
			if reads == 2 {
				object := d.Objects[accountID]
				object.Attributes = map[string][][]byte{
					"objectguid": object.Attributes["objectguid"], "objectclass": object.Attributes["objectclass"],
					"distinguishedname": {[]byte("CN=moved,DC=unit,DC=invalid")},
				}
				d.Objects[accountID] = object
			}
		}
		return d.ResolveIdentity(ctx, target, options)
	}}
	computer, _ := shared.IdentityByGUID(computerID)
	account, _ := shared.IdentityByGUID(accountID)
	result, err := common.Read(t.Context(), service, computer, account)
	if err == nil || result.Confirmed {
		t.Fatal("a stale account name became stable relationship proof")
	}
}

// TestDiscoveryKeepsEmptyAndHiddenIndeterminate preserves the visibility limit shared by actual empty and hidden account attributes.
func TestDiscoveryKeepsEmptyAndHiddenIndeterminate(t *testing.T) {
	d := directory(t)
	computer, _ := shared.IdentityByGUID(computerID)
	empty, err := common.Discover(t.Context(), d, computer, 10)
	if !errors.Is(err, common.ErrVisibility) || empty.Complete || len(empty.Edges) != 0 {
		t.Fatal("an omitted account attribute was treated as proof of an empty computer", err)
	}
	service := &mocks.Service{Links: d.ComputerServiceAccounts, Resolve: d.ResolveIdentity, Assert: func(context.Context, shared.SearchOptions) (shared.SearchResult, error) {
		return shared.SearchResult{Complete: true}, nil
	}}
	unknown, err := common.Discover(t.Context(), service, computer, 10)
	if err == nil || unknown.Complete {
		t.Fatal("omitted account attribute and zero search rows became an empty computer")
	}
	d.SetLink(computerID, otherID, true)
	d.SetLink(computerID, accountID, true)
	all, err := common.Discover(t.Context(), d, computer, 10)
	if err != nil || !all.Complete || len(all.Edges) != 2 || all.Edges[0].Account.GUID.String() != accountID {
		t.Fatal("direct links were not independently resolved and sorted", err)
	}
	bounded, err := common.Discover(t.Context(), d, computer, 1)
	if err == nil || bounded.Complete || !bounded.Bounded || len(bounded.Edges) != 1 {
		t.Fatal("bounded direct association discovery claimed completion", err)
	}
}

// TestInvalidPairsAndBoundsRejectUnknownValues keeps malformed identities and unbounded discovery out of the SDK.
func TestInvalidPairsAndBoundsRejectUnknownValues(t *testing.T) {
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("CN=name,DC=unit"), types.StringValue("00000000-0000-0000-0000-000000000000")} {
		_, _, diagnostics := common.ConstructPair(types.StringValue(computerID), value)
		if !diagnostics.HasError() {
			t.Fatal("invalid account GUID was accepted", value)
		}
	}
	for _, limit := range []types.Int64{types.Int64Unknown(), types.Int64Value(0), types.Int64Value(-1), types.Int64Value(10001)} {
		_, _, diagnostics := common.ConstructQuery(common.QueryModel{ComputerGUID: types.StringValue(computerID), ResultLimit: limit})
		if !diagnostics.HasError() {
			t.Fatal("unknown or unbounded association discovery was accepted", limit)
		}
	}
}

// TestDiscoveryUsesExplicitComputerMetadata exercises the SDK range contract, which does not select a distinguishedName attribute.
func TestDiscoveryUsesExplicitComputerMetadata(t *testing.T) {
	d := directory(t)
	d.SetLink(computerID, accountID, true)
	service := &mocks.Service{
		Resolve: d.ResolveIdentity,
		Assert:  d.Search,
		Links: func(ctx context.Context, target shared.Identity, options sdkaccounts.ComputerServiceAccountsOptions) (sdkaccounts.ComputerServiceAccountsResult, error) {
			result, err := d.ComputerServiceAccounts(ctx, target, options)
			if err != nil {
				return result, fmt.Errorf("read synthetic SDK range: %w", err)
			}
			original := *result.Computer.Object
			original.Attributes = map[string][][]byte{
				"objectguid": original.Attributes["objectguid"], "objectclass": original.Attributes["objectclass"],
			}
			result.Computer.Object = &original
			result.Computer.Candidates = []shared.Object{original}
			return result, nil
		},
	}
	computer, _ := shared.IdentityByGUID(computerID)
	observed, err := common.Discover(t.Context(), service, computer, 10)
	if err != nil || !observed.Complete || len(observed.Edges) != 1 || observed.Computer.DN != "CN=computer,DC=unit,DC=invalid" {
		t.Fatal("SDK range metadata was treated as a guarantee of explicitly selected computer properties", err)
	}
}

// TestHiddenValuesCannotBecomeNegativeProof reproduces AD's hidden-value filter semantics from actual native qualification.
func TestHiddenValuesCannotBecomeNegativeProof(t *testing.T) {
	d := directory(t)
	d.SetLink(computerID, accountID, true)
	service := &mocks.Service{
		Resolve: d.ResolveIdentity,
		Links: func(ctx context.Context, target shared.Identity, options sdkaccounts.ComputerServiceAccountsOptions) (sdkaccounts.ComputerServiceAccountsResult, error) {
			result, err := d.ComputerServiceAccounts(ctx, target, options)
			if err != nil {
				return result, fmt.Errorf("read synthetic hidden range: %w", err)
			}
			result.Links = nil
			result.AttributeReturned = false
			return result, nil
		},
		Assert: func(_ context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
			filter, _ := options.TypedFilter.LDAP()
			if strings.Contains(filter, "(!") {
				return shared.SearchResult{Objects: []shared.Object{d.Objects[computerID]}, Complete: true}, nil
			}
			return shared.SearchResult{Complete: true}, nil
		},
	}
	computer, _ := shared.IdentityByGUID(computerID)
	account, _ := shared.IdentityByGUID(accountID)
	edge, err := common.Read(t.Context(), service, computer, account)
	if err == nil || edge.Confirmed {
		t.Fatal("a hidden existing edge became confirmed absent")
	}
	observed, err := common.Discover(t.Context(), service, computer, 10)
	if err == nil || observed.Complete {
		t.Fatal("a hidden nonempty computer became a complete empty collection")
	}
}

// TestAssociationEndpointsRequireComputerAndManagedAccount rejects ordinary users and reversed endpoint classes before any write.
func TestAssociationEndpointsRequireComputerAndManagedAccount(t *testing.T) {
	for _, class := range []string{"user", "group", "computer", "foreignSecurityPrincipal"} {
		d := directory(t)
		d.SetLink(computerID, accountID, true)
		object := d.Objects[accountID]
		object.Attributes["objectclass"] = [][]byte{[]byte(class)}
		d.Objects[accountID] = object
		computer, _ := shared.IdentityByGUID(computerID)
		account, _ := shared.IdentityByGUID(accountID)
		if edge, err := common.Read(t.Context(), d, computer, account); err == nil || edge.Confirmed {
			t.Fatal("non-managed account class became an importable association", class)
		}
		if found, err := common.Discover(t.Context(), d, computer, 10); err == nil || found.Complete || len(found.Edges) != 0 {
			t.Fatal("non-managed account class escaped discovery checks", class)
		}
	}
	for _, class := range []string{"msDS-ManagedServiceAccount", "msDS-GroupManagedServiceAccount"} {
		d := directory(t)
		d.SetLink(computerID, accountID, true)
		object := d.Objects[accountID]
		object.Attributes["objectclass"] = [][]byte{[]byte(class)}
		d.Objects[accountID] = object
		computer, _ := shared.IdentityByGUID(computerID)
		account, _ := shared.IdentityByGUID(accountID)
		if edge, err := common.Read(t.Context(), d, computer, account); err != nil || !edge.Confirmed || !edge.Present {
			t.Fatal("supported managed service-account class was rejected", class, err)
		}
	}
}
