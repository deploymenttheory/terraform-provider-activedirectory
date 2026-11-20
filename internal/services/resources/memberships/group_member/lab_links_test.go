// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// setLink introduces one known fixture link or independent drift only after rechecking both endpoint boundaries.
func (l *groupMemberLab) setLink(t *testing.T, group, member labSubject, present bool) {
	t.Helper()
	if group.Kind != "group" {
		t.Fatal("membership fixture owner is not a group")
	}
	for _, subject := range []labSubject{group, member} {
		if _, err := l.ownedObject(t.Context(), subject); err != nil {
			t.Fatal("fixture ownership did not verify before linked-value mutation")
		}
	}
	var result memberships.GroupMembershipMutationResult
	var err error
	if present {
		result, err = l.observer.Memberships.AddGroupMembers(t.Context(), memberships.AddGroupMembersRequest{
			Identity: l.identity(group), Members: []shared.Identity{l.identity(member)}, Mode: memberships.GroupMembershipStrict,
		})
	} else {
		result, err = l.observer.Memberships.RemoveGroupMembers(t.Context(), memberships.RemoveGroupMembersRequest{
			Identity: l.identity(group), Members: []shared.Identity{l.identity(member)}, Mode: memberships.GroupMembershipStrict,
		})
	}
	if err != nil || result.Mutation.State != shared.ObjectWriteAccepted || result.Mutation.TargetGUID == nil || result.Mutation.TargetGUID.String() != group.GUID {
		t.Fatal("independent direct-link mutation was not acknowledged; no automatic retry is authorized")
	}
	if err := l.assertLink(group, member, present); err != nil {
		t.Fatal("independent direct-link fixture readback did not verify")
	}
}

// assertLink observes direct values through the independent administrative fixture session, separate from provider assertion mapping.
func (l *groupMemberLab) assertLink(group, member labSubject, present bool) error {
	if _, err := l.ownedObject(context.Background(), group); err != nil {
		return errLabOwnership
	}
	object, err := l.ownedObject(context.Background(), member)
	if err != nil {
		return errLabOwnership
	}
	observed, err := l.observer.Memberships.GroupMemberLinks(context.Background(), l.identity(group), memberships.GroupMemberLinksOptions{MaxValues: 20})
	if err != nil || !observed.Complete || observed.Version == nil || observed.Version.GUID.String() != group.GUID {
		return errLabOwnership
	}
	found := false
	for _, link := range observed.Members {
		if strings.EqualFold(link.DistinguishedName, object.DistinguishedName) {
			found = true
		}
	}
	if found != present {
		return errLabOwnership
	}
	return nil
}
