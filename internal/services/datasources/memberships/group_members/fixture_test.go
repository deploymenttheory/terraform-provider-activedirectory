// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_members/mocks"
)

const (
	rootID     = "11111111-1111-4111-8111-111111111111"
	userID     = "22222222-2222-4222-8222-222222222222"
	nestedID   = "33333333-3333-4333-8333-333333333333"
	computerID = "44444444-4444-4444-8444-444444444444"
)

var (
	errSyntheticRequest = errors.New("unexpected synthetic membership query")
	errPrivatePeer      = errors.New("private-peer-password-must-not-be-exposed")
)

type queryFixture struct {
	objects map[string]shared.Object
	queries int
}

// loadFixture reads feature-local response objects and encodes their real binary GUID/SID attribute representations.
func loadFixture(t *testing.T) *queryFixture {
	t.Helper()
	data, err := os.ReadFile("tests/responses/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		GUID  string `json:"guid"`
		Name  string `json:"name"`
		DN    string `json:"dn"`
		Class string `json:"class"`
		SID   string `json:"sid"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	fixture := &queryFixture{objects: map[string]shared.Object{}}
	for _, row := range rows {
		guid, guidErr := shared.ParseGUID(row.GUID)
		sid, sidErr := shared.ParseSID(row.SID)
		if guidErr != nil || sidErr != nil {
			t.Fatal("invalid synthetic principal identity")
		}
		fixture.objects[row.GUID] = shared.Object{DistinguishedName: row.DN, Attributes: map[string][][]byte{
			"objectguid": {guid.Bytes()}, "objectsid": {sid.Bytes()}, "distinguishedname": {[]byte(row.DN)},
			"objectclass": {[]byte("top"), []byte(row.Class)}, "name": {[]byte(row.Name)},
			"samaccountname": {[]byte(row.Name)}, "usnchanged": {[]byte("1")},
		}}
	}
	fixture.objects[userID].Attributes["primarygroupid"] = [][]byte{[]byte("1001")}
	return fixture
}

// resolve refuses a different scope or projection and returns one complete, matching group identity.
func (f *queryFixture) resolve(_ context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	if target.Form() != shared.IdentityFormGUID || target.String() != rootID || len(options.Properties) != 3 ||
		!slices.Contains(options.Properties, "distinguishedName") || !slices.Contains(options.Properties, "objectClass") || !slices.Contains(options.Properties, "objectGUID") {
		return shared.IdentityResult{}, errSyntheticRequest
	}
	object := f.objects[rootID]
	return shared.IdentityResult{Object: &object, Candidates: []shared.Object{object}, Complete: true}, nil
}

// principal constructs a typed SDK response over the independently decoded fixture attributes.
func (f *queryFixture) principal(id string, primary bool) memberships.MembershipPrincipal {
	object := f.objects[id]
	guid, _ := shared.ParseGUID(id)
	value, _ := object.Values("objectSid").Single()
	sid, _ := value.SID()
	return memberships.MembershipPrincipal{Object: object, GUID: guid, SID: sid, Source: config.Endpoint{Host: "dc.unit.invalid", Port: 636}, Primary: primary}
}

// node retains realistic range-anchor metadata without implementing the SDK's traversal algorithm in the mock.
func (f *queryFixture) node(id string, depth int, members ...string) memberships.GroupMembershipTraversal {
	object := f.objects[id]
	guid, _ := shared.ParseGUID(id)
	links := memberships.GroupMemberLinksResult{
		Observation: shared.IdentityResult{Object: &object, Candidates: []shared.Object{object}, Complete: true},
		Version:     &shared.ObjectVersion{GUID: guid, USNChanged: 1}, Complete: true, AttributeReturned: len(members) > 0,
		Ranges: []memberships.GroupMemberRange{{RequestedProperty: "member;range=0-1499"}},
	}
	for _, member := range members {
		dn := f.objects[member].DistinguishedName
		links.Members = append(links.Members, memberships.GroupMemberLink{DistinguishedName: dn, Raw: dn})
	}
	return memberships.GroupMembershipTraversal{Source: config.Endpoint{Host: "dc.unit.invalid", Port: 636}, Depth: depth, Links: links}
}

// result provides explicit SDK observation scenarios, including duplicate paths, cycle-safe traversal and primary-only membership.
func (f *queryFixture) result(mode string) memberships.GroupMembersResult {
	result := memberships.GroupMembersResult{Groups: []memberships.GroupMembershipTraversal{f.node(rootID, 0, userID, nestedID)}, Complete: true}
	switch mode {
	case "recursive":
		result.Groups = append(result.Groups, f.node(nestedID, 1, rootID, userID, computerID))
		result.Members = []memberships.MembershipPrincipal{f.principal(computerID, false), f.principal(userID, false), f.principal(userID, true)}
	case "primary":
		result.Groups = []memberships.GroupMembershipTraversal{f.node(rootID, 0)}
		result.Groups[0].PrimaryMembers = shared.SearchResult{Objects: []shared.Object{f.objects[userID]}, Complete: true}
		result.Members = []memberships.MembershipPrincipal{f.principal(userID, true)}
	case "hidden":
		result.Groups = []memberships.GroupMembershipTraversal{f.node(rootID, 0)}
	case "bounded":
		result.Members = []memberships.MembershipPrincipal{f.principal(userID, true)}
		result.Complete, result.LimitReached = false, true
	default:
		result.Members = []memberships.MembershipPrincipal{f.principal(nestedID, false), f.principal(userID, true)}
	}
	return result
}

// service validates the request boundary and returns only the selected explicit response scenario.
func (f *queryFixture) service(mode string) *mocks.Service {
	return &mocks.Service{
		Resolve: f.resolve,
		Query: func(_ context.Context, target shared.Identity, options memberships.GroupMembersOptions) (memberships.GroupMembersResult, error) {
			f.queries++
			if target.String() != rootID || !slices.Contains(options.Properties, "distinguishedName") || len(options.Contexts) != 0 ||
				options.Limits.MaxObjects < 1 || options.Limits.MaxGroups < 1 || options.Limits.MaxRanges < 1 || options.Limits.MaxDepth < 1 || options.Limits.MaxBytes < 1024 {
				return memberships.GroupMembersResult{}, errSyntheticRequest
			}
			selected := mode
			if selected == "" {
				selected = "direct"
				if options.Recursive {
					selected = "recursive"
				}
			}
			result := f.result(selected)
			if selected == "bounded" {
				return result, errors.Join(memberships.ErrMembershipLimit, nil)
			}
			return result, nil
		},
	}
}

// fixtureConfig loads the feature's actual Terraform CLI configuration before a test changes its working directory.
func fixtureConfig(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "__RECURSIVE__", "false")
}
