// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/groups/group/mocks"
)

const fixtureGUID = "b836f3d7-6ab2-45f4-a4db-67827ee3be44"

// unitDirectory is the synthetic public-SDK seam, not a replacement implementation of LDAP or server conversion rules.
type unitDirectory struct {
	mu       sync.Mutex
	GUID     string `json:"guid"`
	Name     string `json:"name"`
	ParentDN string `json:"parentDn"`
	flags    shared.Flags32
	text     map[string]string
	exists   bool
	calls    map[string]int
}

func newUnitDirectory(t *testing.T) *unitDirectory {
	t.Helper()
	content, err := os.ReadFile("tests/responses/group.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := &unitDirectory{flags: 0x40000000, text: map[string]string{}, calls: map[string]int{}}
	if err := json.Unmarshal(content, directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func (d *unitDirectory) snapshot() *groups.Group {
	guid, _ := shared.ParseGUID(d.GUID)
	name, dn, flags := d.Name, "CN="+d.Name+","+d.ParentDN, d.flags
	category, scope := groups.GroupDistribution, groups.GroupScopeUnknown
	if flags&0x80000000 != 0 {
		category = groups.GroupSecurity
	}
	switch flags & (2 | 4 | 8) {
	case 2:
		scope = groups.GroupGlobal
	case 4:
		scope = groups.GroupDomainLocal
	case 8:
		scope = groups.GroupUniversal
	}
	observed := &groups.Group{
		GUID: &guid, Name: &name, DistinguishedName: dn, Type: &flags, Category: &category, Scope: &scope,
		Raw: shared.Object{DistinguishedName: dn, Attributes: map[string][][]byte{
			"objectguid": {guid.Bytes()}, "objectclass": {[]byte("top"), []byte("group")}, "name": {[]byte(name)}, "grouptype": {[]byte(flags.LDAPString())},
		}},
	}
	for ldap, target := range map[string]**string{
		"sAMAccountName": &observed.SAMAccountName, "description": &observed.Description, "displayName": &observed.DisplayName,
		"wWWHomePage": &observed.HomePage, "managedBy": &observed.ManagedByDN,
	} {
		observed.Raw.RequestedProperties = append(observed.Raw.RequestedProperties, ldap)
		if text, exists := d.text[strings.ToLower(ldap)]; exists {
			value := text
			*target = &value
			observed.Raw.Attributes[strings.ToLower(ldap)] = [][]byte{[]byte(text)}
		}
	}
	return observed
}

func (d *unitDirectory) mutation() shared.ObjectMutationResult {
	observed := d.snapshot()
	return shared.ObjectMutationResult{
		State: shared.ObjectWriteAccepted, TargetGUID: observed.GUID, TargetDN: observed.DistinguishedName,
		After: &observed.Raw, Verified: true, ReadbackComplete: true,
	}
}

func (d *unitDirectory) setProperties(request groups.UpdateGroupRequest) {
	for ldap, value := range map[string]*string{
		"samaccountname": request.SAMAccountName, "description": request.Description, "displayname": request.DisplayName,
		"wwwhomepage": request.HomePage, "managedby": request.ManagedByDN,
	} {
		if value != nil {
			d.text[ldap] = *value
		}
	}
	for _, change := range request.Changes {
		if change.Kind == shared.AttributeClear {
			delete(d.text, strings.ToLower(change.Attribute))
		}
	}
	if request.Category != nil {
		d.flags &^= 0x80000000
		if *request.Category == groups.GroupSecurity {
			d.flags |= 0x80000000
		}
	}
	if request.Scope != nil {
		d.flags &^= 2 | 4 | 8
		switch *request.Scope {
		case groups.GroupGlobal:
			d.flags |= 2
		case groups.GroupDomainLocal:
			d.flags |= 4
		case groups.GroupUniversal:
			d.flags |= 8
		}
	}
}

func (d *unitDirectory) callbacks(t *testing.T) *featuremocks.Service {
	t.Helper()
	checkTarget := func(target shared.Identity, options shared.ObjectMutationOptions) {
		if target.Form() != shared.IdentityFormGUID || target.String() != d.GUID || options.ExpectedVersion != nil {
			t.Error("group operation lost GUID identity or introduced an ordinary-write assertion")
		}
	}
	return &featuremocks.Service{
		Read: func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (groups.GroupResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(target, shared.ObjectMutationOptions{})
			if !d.exists {
				return groups.GroupResult{}, shared.ErrDirectoryObjectNotFound
			}
			observed := d.snapshot()
			return groups.GroupResult{Group: observed, Observation: shared.IdentityResult{Complete: true, Object: &observed.Raw}}, nil
		},
		Create: func(_ context.Context, request groups.CreateGroupRequest) (groups.GroupMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return groups.GroupMutationResult{}, shared.ErrObjectAlreadyExists
			}
			if request.Parent.Form() != shared.IdentityFormDN || request.Options.ExpectedVersion != nil || len(request.Attributes) != 0 {
				t.Error("group creation guessed a parent, introduced an assertion or submitted raw flags/membership")
			}
			d.Name, d.ParentDN, d.exists = request.Name, request.Parent.String(), true
			d.setProperties(groups.UpdateGroupRequest{
				Category: &request.Category, Scope: &request.Scope, SAMAccountName: request.SAMAccountName,
				Description: request.Description, DisplayName: request.DisplayName, HomePage: request.HomePage, ManagedByDN: request.ManagedByDN,
			})
			return groups.GroupMutationResult{Group: d.snapshot(), Mutation: d.mutation()}, nil
		},
		Update: func(_ context.Context, request groups.UpdateGroupRequest) (groups.GroupMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(request.Identity, request.Options)
			d.calls["update"]++
			for _, change := range request.Changes {
				if !strings.EqualFold(change.Attribute, "description") || change.Kind != shared.AttributeClear {
					t.Error("group update submitted unexpected raw attributes instead of typed patches")
				}
			}
			d.setProperties(request)
			result := groups.GroupMutationResult{Group: d.snapshot(), Mutation: d.mutation()}
			if len(request.Changes) != 0 {
				result.Mutation.Verified = false
				return result, shared.ErrObjectUnverifiable
			}
			return result, nil
		},
		Rename: func(_ context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(request.Identity, request.Options)
			d.calls["rename"]++
			d.Name = request.Name
			return d.mutation(), nil
		},
		Move: func(_ context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(request.Identity, request.Options)
			if request.Destination != nil || request.TargetParent.Form() != shared.IdentityFormDN {
				t.Error("ordinary group move enabled cross-domain behavior or guessed a parent")
			}
			d.calls["move"]++
			d.ParentDN = request.TargetParent.String()
			return shared.ObjectMoveResult{Mutation: d.mutation()}, nil
		},
		Delete: func(_ context.Context, request groups.DeleteGroupRequest) (groups.GroupMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(request.Identity, request.Options)
			d.calls["delete"]++
			result := groups.GroupMutationResult{Group: d.snapshot(), Mutation: d.mutation()}
			tombstone := d.snapshot().Raw
			tombstone.Attributes["isdeleted"] = [][]byte{[]byte("TRUE")}
			result.Mutation.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}}
			d.exists = false
			return result, nil
		},
	}
}
