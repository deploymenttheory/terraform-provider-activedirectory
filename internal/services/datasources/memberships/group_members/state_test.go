// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestProjectionRetainsUnresolvedForeignReferencesWithoutInventingPrincipals preserves a usable row and a separate unresolved DN.
func TestProjectionRetainsUnresolvedForeignReferencesWithoutInventingPrincipals(t *testing.T) {
	f := loadFixture(t)
	group, options, diagnostics := constructQuery(GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID)})
	if diagnostics.HasError() || options.Limits.MaxObjects != 1000 || options.Limits.MaxGroups != 100 || options.Limits.MaxRanges != 100 ||
		options.Limits.MaxDepth != 8 || options.Limits.MaxBytes != 8<<20 || len(options.Contexts) != 0 {
		t.Fatal("absent bounds did not construct the explicit finite single-context defaults", diagnostics)
	}
	result := f.result("bounded")
	result.Unresolved = []memberships.MembershipResolution{{
		Reference: "CN=S-1-5-21-9-9-9-99,CN=ForeignSecurityPrincipals,DC=unit,DC=invalid", Error: memberships.ErrMembershipContext,
	}}
	projected := projectResult(result, group, options)
	if projected.complete || len(projected.members) != 1 || len(projected.unresolved) != 1 {
		t.Fatal("partial foreign membership was lost or presented as complete")
	}
	unresolved := projected.unresolved[0].(types.Object).Attributes()
	if _, fabricated := unresolved["id"]; fabricated {
		t.Fatal("an unresolved foreign reference became a principal GUID")
	}
}

// TestTraversalContradictionCannotPublishWrongScopeMembers refuses contradictory positive group metadata before row projection.
func TestTraversalContradictionCannotPublishWrongScopeMembers(t *testing.T) {
	f := loadFixture(t)
	group, options, _ := constructQuery(GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID)})
	for _, mode := range []string{"wrong-root", "wrong-version", "wrong-candidate", "unexpected-recursion"} {
		result := f.result("direct")
		switch mode {
		case "wrong-root":
			result.Groups[0] = f.node(nestedID, 0, userID)
		case "wrong-version":
			guid, _ := shared.ParseGUID(nestedID)
			result.Groups[0].Links.Version.GUID = guid
		case "wrong-candidate":
			result.Groups[0].Links.Observation.Candidates = []shared.Object{f.objects[nestedID]}
		case "unexpected-recursion":
			result.Groups = append(result.Groups, f.node(nestedID, 1, userID))
		}
		projected := projectResult(result, group, options)
		if projected.complete || !projected.diagnostics.HasError() || len(projected.members) != 0 {
			t.Fatal("contradictory traversal produced usable members for the requested scope", mode)
		}
	}
}

// TestKnownRangeFailurePreservesUsablePrincipals retains typed rows while refusing a false SDK completion flag.
func TestKnownRangeFailurePreservesUsablePrincipals(t *testing.T) {
	f := loadFixture(t)
	group, options, _ := constructQuery(GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID)})
	result := f.result("direct")
	result.Groups[0].Links.Ranges[0].Error = errPrivatePeer
	projected := projectResult(result, group, options)
	if projected.complete || !projected.diagnostics.HasError() || len(projected.members) != 2 {
		t.Fatal("range failure discarded usable observations or became complete")
	}
}

// TestConflictingDuplicateMetadataIsNotArbitrarilySelected drops both conflicting versions instead of depending on response order.
func TestConflictingDuplicateMetadataIsNotArbitrarilySelected(t *testing.T) {
	f := loadFixture(t)
	group, options, _ := constructQuery(GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID)})
	for _, reverse := range []bool{false, true} {
		first := f.principal(userID, true)
		second := f.principal(userID, false)
		attributes := make(map[string][][]byte, len(second.Object.Attributes))
		for name, values := range second.Object.Attributes {
			attributes[name] = values
		}
		attributes["distinguishedname"] = [][]byte{[]byte("CN=renamed,DC=unit,DC=invalid")}
		second.Object.Attributes = attributes
		result := f.result("direct")
		result.Members = []memberships.MembershipPrincipal{first, second}
		if reverse {
			result.Members[0], result.Members[1] = result.Members[1], result.Members[0]
		}
		projected := projectResult(result, group, options)
		if projected.complete || !projected.diagnostics.HasError() || len(projected.members) != 0 {
			t.Fatal("conflicting duplicate identity selected a response-order-dependent name")
		}
	}
}
