// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var errRequest = errors.New("unexpected synthetic direct membership request")

// Directory models only direct links over explicit synthetic objects, without contacting AD or inferring effective membership.
type Directory struct {
	mu      sync.Mutex
	Objects map[string]shared.Object
	Links   map[string]map[string]bool
	Adds    int
	Removes int
}

// DecodeDirectory loads feature-local object fixtures with actual binary GUID attributes.
func DecodeDirectory(data []byte) (*Directory, error) {
	var objects []struct {
		GUID  string `json:"guid"`
		DN    string `json:"dn"`
		Class string `json:"class"`
	}
	if err := json.Unmarshal(data, &objects); err != nil {
		return nil, fmt.Errorf("decode synthetic membership objects: %w", err)
	}
	directory := &Directory{Objects: map[string]shared.Object{}, Links: map[string]map[string]bool{}}
	for _, object := range objects {
		guid, err := shared.ParseGUID(object.GUID)
		if err != nil {
			return nil, fmt.Errorf("decode synthetic member GUID: %w", err)
		}
		directory.Objects[guid.String()] = shared.Object{DistinguishedName: object.DN, Attributes: map[string][][]byte{
			"objectguid": {guid.Bytes()}, "distinguishedname": {[]byte(object.DN)}, "objectclass": {[]byte("top"), []byte(object.Class)},
		}}
	}
	return directory, nil
}

// ResolveIdentity returns only an explicitly matching synthetic GUID or DN and complete unique lookup metadata.
func (d *Directory) ResolveIdentity(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for id, object := range d.Objects {
		if (target.Form() == shared.IdentityFormGUID && target.String() == id) ||
			(target.Form() == shared.IdentityFormDN && strings.EqualFold(target.String(), object.DistinguishedName)) {
			return shared.IdentityResult{Object: &object, Candidates: []shared.Object{object}, Complete: true}, nil
		}
	}
	return shared.IdentityResult{}, shared.ErrDirectoryObjectNotFound
}

// Search evaluates the feature's explicit positive/negative direct-member assertion over the synthetic group.
func (d *Directory) Search(_ context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	filter, err := options.TypedFilter.LDAP()
	if err != nil || options.Scope != shared.ScopeBase || options.PageSize != 2 || options.ResultLimit != 2 {
		return shared.SearchResult{}, errRequest
	}
	for id, object := range d.Objects {
		if object.DistinguishedName != options.Base {
			continue
		}
		matched := false
		if strings.Contains(filter, "(!(member=*))") {
			matched = len(d.Links[id]) == 0
		} else {
			for member, candidate := range d.Objects {
				assertion, _ := shared.FilterEqual("member", shared.FilterText(candidate.DistinguishedName)).LDAP()
				if strings.Contains(filter, assertion) {
					matched = d.Links[id][member]
					if strings.Contains(filter, "(!"+assertion+")") {
						matched = !matched
					}
					break
				}
			}
		}
		if matched {
			return shared.SearchResult{Objects: []shared.Object{object}, Complete: true}, nil
		}
		return shared.SearchResult{Complete: true}, nil
	}
	return shared.SearchResult{}, shared.ErrDirectoryObjectNotFound
}

// GroupMemberLinks returns only configured direct links, honoring the selected value bound.
func (d *Directory) GroupMemberLinks(_ context.Context, target shared.Identity, options memberships.GroupMemberLinksOptions) (memberships.GroupMemberLinksResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	object, ok := d.Objects[target.String()]
	if !ok {
		return memberships.GroupMemberLinksResult{}, shared.ErrDirectoryObjectNotFound
	}
	guid, _ := shared.ParseGUID(target.String())
	result := memberships.GroupMemberLinksResult{
		Observation: shared.IdentityResult{Object: &object, Candidates: []shared.Object{object}, Complete: true},
		Version:     &shared.ObjectVersion{GUID: guid}, Complete: true, AttributeReturned: len(d.Links[target.String()]) > 0,
		Ranges: []memberships.GroupMemberRange{{RequestedProperty: "member;range=0-*"}},
	}
	ids := []string{}
	for member := range d.Links[target.String()] {
		ids = append(ids, member)
	}
	slices.Sort(ids)
	for _, member := range ids {
		if len(result.Members) == options.MaxValues {
			result.Complete = false
			return result, memberships.ErrGroupMembershipLimit
		}
		dn := d.Objects[member].DistinguishedName
		result.Members = append(result.Members, memberships.GroupMemberLink{DistinguishedName: dn, Raw: dn})
	}
	return result, nil
}

// AddGroupMembers accepts exactly one persistent GUID pair and preserves all other configured links.
func (d *Directory) AddGroupMembers(_ context.Context, request memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(request.Members) != 1 || request.TimeToLive != nil || request.Mode != memberships.GroupMembershipPermissive {
		return memberships.GroupMembershipMutationResult{}, errRequest
	}
	d.Adds++
	return d.change(request.Identity, request.Members[0], true), nil
}

// RemoveGroupMembers removes exactly one configured direct link without deleting either endpoint.
func (d *Directory) RemoveGroupMembers(_ context.Context, request memberships.RemoveGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(request.Members) != 1 || request.Mode != memberships.GroupMembershipPermissive {
		return memberships.GroupMembershipMutationResult{}, errRequest
	}
	d.Removes++
	return d.change(request.Identity, request.Members[0], false), nil
}

func (d *Directory) change(group, member shared.Identity, present bool) memberships.GroupMembershipMutationResult {
	if d.Links[group.String()] == nil {
		d.Links[group.String()] = map[string]bool{}
	}
	if present {
		d.Links[group.String()][member.String()] = true
	} else {
		delete(d.Links[group.String()], member.String())
	}
	guid, _ := shared.ParseGUID(group.String())
	return memberships.GroupMembershipMutationResult{
		Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid},
		Members:  []memberships.GroupMemberInput{{Identity: member}},
	}
}

// SetLink applies independent unit-fixture drift without invoking the provider's mutation callbacks.
func (d *Directory) SetLink(group, member string, present bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	groupIdentity, _ := shared.IdentityByGUID(group)
	memberIdentity, _ := shared.IdentityByGUID(member)
	d.change(groupIdentity, memberIdentity, present)
}

// HasLink independently checks the synthetic direct link after Terraform operations.
func (d *Directory) HasLink(group, member string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Links[group][member]
}
