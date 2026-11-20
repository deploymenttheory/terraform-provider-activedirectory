// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sort"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// createSubject independently provisions an owned endpoint; account fixture passwords never enter Terraform configuration.
func (l *groupMembersLab) createSubject(t *testing.T, kind string) labSubject {
	return l.createNamedSubject(t, kind, kind)
}

// createNamedSubject provisions one distinctly named owned fixture without reusing an existing account identity.
func (l *groupMembersLab) createNamedSubject(t *testing.T, kind, label string) labSubject {
	t.Helper()
	parent, _ := shared.IdentityByGUID(l.parentGUID)
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("fixture password generation failed")
	}
	defer clear(random)
	password := []byte("Aa1!" + base64.RawStdEncoding.EncodeToString(random))
	l.secrets = append(l.secrets, string(password))
	defer clear(password)
	name, sam, marker, disabled := l.prefix+"-"+label, l.sam(label), l.marker, false
	var mutation shared.ObjectMutationResult
	var err error
	switch kind {
	case "group":
		result, createErr := l.observer.Groups.CreateGroup(t.Context(), groups.CreateGroupRequest{
			Parent: parent, Name: name, Category: groups.GroupSecurity, Scope: groups.GroupGlobal, SAMAccountName: &sam, Description: &marker,
		})
		mutation, err = result.Mutation, createErr
		if err == nil && !result.Mutation.Verified {
			err = errLabOwnership
		}
	case "user":
		result, createErr := l.observer.Users.CreateUser(
			t.Context(),
			users.CreateUserRequest{
				Parent:          parent,
				Name:            name,
				InitialPassword: password,
				Properties:      users.UserProperties{SAMAccountName: &sam, Description: &marker, Enabled: &disabled},
			},
		)
		mutation, err = result.Creation, createErr
		if err == nil && !result.Complete {
			err = errLabOwnership
		}
	case "computer":
		result, createErr := l.observer.Computers.CreateComputer(
			t.Context(),
			computers.CreateComputerRequest{
				Parent:          parent,
				Name:            name,
				InitialPassword: password,
				Properties:      computers.ComputerProperties{SAMAccountName: &sam, Description: &marker},
				Control:         accounts.AccountControlPatch{Enabled: &disabled},
			},
		)
		mutation, err = result.Creation, createErr
		if err == nil && !result.Complete {
			err = errLabOwnership
		}
	default:
		t.Fatal("unsupported owned account fixture class")
	}
	subject := labSubject{Kind: kind}
	if mutation.TargetGUID != nil {
		subject.GUID = mutation.TargetGUID.String()
		l.subjects[subject.GUID] = subject
	}
	if err != nil || mutation.State != shared.ObjectWriteAccepted || subject.GUID == "" {
		t.Fatal("independent account fixture creation did not complete; preserve known fixture identity")
	}
	if _, err := l.ownedObject(t.Context(), subject); err != nil {
		t.Fatal("created account ownership did not independently verify")
	}
	return subject
}

// renameSubject moves no ownership boundary and verifies the renamed fixture still has the original GUID and marker.
func (l *groupMembersLab) renameSubject(t *testing.T, subject labSubject) {
	t.Helper()
	if _, err := l.ownedObject(t.Context(), subject); err != nil {
		t.Fatal("fixture ownership did not verify before rename")
	}
	result, err := l.observer.Directory.RenameObject(t.Context(), shared.RenameObjectRequest{Identity: l.identity(subject), Name: l.prefix + "-renamed-" + subject.Kind})
	if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified {
		t.Fatal("external fixture rename did not verify")
	}
	if _, err := l.ownedObject(t.Context(), subject); err != nil {
		t.Fatal("same-GUID renamed fixture ownership did not verify")
	}
}

// retire deletes only independently owned leaf fixtures after Terraform relinquishment, then proves exact GUID absence.
func (l *groupMembersLab) retire(ctx context.Context) error {
	subjects := make([]labSubject, 0, len(l.subjects))
	for _, subject := range l.subjects {
		subjects = append(subjects, subject)
	}
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].GUID < subjects[j].GUID })
	for _, subject := range subjects {
		target := l.identity(subject)
		observed, readErr := l.observer.Directory.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{})
		if lifecycle.ConfirmedAbsent(observed, readErr) {
			continue
		}
		object, err := l.ownedObject(ctx, subject)
		if err != nil {
			return errLabOwnership
		}
		children, err := l.observer.Directory.Search(
			ctx,
			shared.SearchOptions{Base: object.DistinguishedName, Scope: shared.ScopeOneLevel, Filter: "(objectClass=*)", ResultLimit: 2},
		)
		if err != nil || !children.Complete || children.LimitReached || len(children.Objects) != 0 {
			return errLabOwnership
		}
		deleted, err := l.observer.Directory.DeleteObject(ctx, shared.DeleteObjectRequest{Identity: target, Recursive: false})
		if !lifecycle.ConfirmedDeletion(deleted, err) {
			return errLabOwnership
		}
	}
	return nil
}
