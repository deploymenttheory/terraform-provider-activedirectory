// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestAccGroupMembersSDKContract isolates the public SDK's actual empty-group traversal before Terraform state mapping or parallel queries.
func TestAccGroupMembersSDKContract(t *testing.T) {
	lab := newGroupMembersLab(t)
	group := lab.createNamedSubject(t, "group", "contract")
	result, err := lab.observer.Memberships.GroupMembers(t.Context(), lab.identity(group), memberships.GroupMembersOptions{
		Properties: []string{"distinguishedName"},
		Limits:     memberships.MembershipLimits{MaxObjects: 100, MaxGroups: 10, MaxRanges: 10, MaxDepth: 4, MaxBytes: 1 << 20},
	})
	var ldap *shared.LDAPError
	code := -1
	if errors.As(err, &ldap) {
		code = int(ldap.Code)
	}
	// Record only typed outcomes and retrieval states. Raw peer errors, attribute values and credentials are never logged.
	fields := map[string]any{
		"error_class": shared.Classify(err), "error_type": fmt.Sprintf("%T", err), "ldap_code": code,
		"groups": len(result.Groups), "members": len(result.Members), "complete": result.Complete, "bounded": result.LimitReached,
		"invalid_search": errors.Is(err, shared.ErrInvalidSearch), "invalid_result": errors.Is(err, shared.ErrInvalidResult),
		"deadline": errors.Is(err, context.DeadlineExceeded), "canceled": errors.Is(err, context.Canceled),
		"context": errors.Is(err, memberships.ErrMembershipContext), "foreign": errors.Is(err, memberships.ErrMembershipForeign),
		"invalid_membership": errors.Is(err, memberships.ErrInvalidMembership),
	}
	if len(result.Groups) > 0 {
		node := result.Groups[0]
		fields["primary_complete"], fields["primary_rows"] = node.PrimaryMembers.Complete, len(node.PrimaryMembers.Objects)
		fields["range_complete"], fields["attribute_returned"] = node.Links.Complete, node.Links.AttributeReturned
		fields["ranges"], fields["resolutions"] = len(node.Links.Ranges), len(node.Resolutions)
		if node.Links.Observation.Object != nil {
			fields["guid_state"] = node.Links.Observation.Object.Values("objectGUID").State
			fields["sid_state"] = node.Links.Observation.Object.Values("objectSid").State
		}
	}
	t.Logf("SDK_CONTRACT %v", fields)
	if err != nil || !result.Complete || len(result.Groups) != 1 || len(result.Members) != 0 {
		t.Fatal("public SDK empty-group traversal failed before provider mapping; raw peer error withheld")
	}
}
