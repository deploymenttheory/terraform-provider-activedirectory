// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT
package group_members

import (
	"context"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// primaryGroup resolves the unchanged primary group of one independently owned fixture user without modifying a built-in group.
func (l *groupMembersLab) primaryGroup(t *testing.T, user labSubject) string {
	t.Helper()
	object, err := l.observer.Directory.ResolveIdentity(t.Context(), l.identity(user), shared.ResolveIdentityOptions{Properties: []string{"objectSid", "primaryGroupID"}})
	if err != nil || !object.Complete || object.Object == nil {
		t.Fatal("fixture primary-group metadata is unavailable")
	}
	value, err := object.Object.Values("objectSid").Single()
	sid, sidErr := value.SID()
	primary, primaryErr := object.Object.Values("primaryGroupID").Single()
	rid, textErr := primary.Text()
	if err != nil || sidErr != nil || primaryErr != nil || textErr != nil || rid != "513" {
		t.Fatal("fresh fixture user does not have the expected unchanged primary group")
	}
	parts := strings.Split(sid.String(), "-")
	primarySID := strings.Join(parts[:len(parts)-1], "-") + "-" + rid
	target, err := shared.IdentityBySID(primarySID)
	if err != nil {
		t.Fatal("fixture primary-group SID is invalid")
	}
	group, err := l.observer.Groups.Group(context.Background(), target, shared.ResolveIdentityOptions{})
	if err != nil || !group.Observation.Complete || group.Group == nil || group.Group.GUID == nil || group.Group.SID == nil || group.Group.SID.String() != primarySID {
		t.Fatal("fixture primary group could not be independently resolved")
	}
	return group.Group.GUID.String()
}
