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

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var errRequest = errors.New("unexpected synthetic direct association request")

// Directory models only direct links over explicit synthetic objects, without contacting AD or inferring effective association.
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
		return nil, fmt.Errorf("decode synthetic association objects: %w", err)
	}
	directory := &Directory{Objects: map[string]shared.Object{}, Links: map[string]map[string]bool{}}
	for _, object := range objects {
		guid, err := shared.ParseGUID(object.GUID)
		if err != nil {
			return nil, fmt.Errorf("decode synthetic account GUID: %w", err)
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

// Search evaluates the feature's explicit positive/negative direct-account assertion over the synthetic computer.
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
		if strings.Contains(filter, "(!(msDS-HostServiceAccount=*))") {
			matched = len(d.Links[id]) == 0
		} else {
			for account, candidate := range d.Objects {
				assertion, _ := shared.FilterEqual("msDS-HostServiceAccount", shared.FilterText(candidate.DistinguishedName)).LDAP()
				if strings.Contains(filter, assertion) {
					matched = d.Links[id][account]
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

// ComputerServiceAccounts returns only configured direct links, honoring the selected value bound.
func (d *Directory) ComputerServiceAccounts(
	_ context.Context,
	target shared.Identity,
	options sdkaccounts.ComputerServiceAccountsOptions,
) (sdkaccounts.ComputerServiceAccountsResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	object, ok := d.Objects[target.String()]
	if !ok {
		return sdkaccounts.ComputerServiceAccountsResult{}, shared.ErrDirectoryObjectNotFound
	}
	guid, _ := shared.ParseGUID(target.String())
	result := sdkaccounts.ComputerServiceAccountsResult{
		Computer: shared.IdentityResult{Object: &object, Candidates: []shared.Object{object}, Complete: true},
		Version:  &shared.ObjectVersion{GUID: guid}, Complete: true, AttributeReturned: len(d.Links[target.String()]) > 0,
		Ranges: []sdkaccounts.ComputerServiceAccountRange{{Property: "msDS-HostServiceAccount;range=0-*"}},
	}
	ids := []string{}
	for account := range d.Links[target.String()] {
		ids = append(ids, account)
	}
	slices.Sort(ids)
	for _, account := range ids {
		if len(result.Links) == options.MaxValues {
			result.Complete = false
			return result, sdkaccounts.ErrServiceAccountAssociationLimit
		}
		dn := d.Objects[account].DistinguishedName
		result.Links = append(result.Links, sdkaccounts.ComputerServiceAccountLink{DistinguishedName: dn})
	}
	return result, nil
}

// AddComputerServiceAccounts accepts exactly one persistent GUID pair and preserves all other configured links.
func (d *Directory) AddComputerServiceAccounts(_ context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(request.Accounts) != 1 || !request.Permissive {
		return sdkaccounts.ComputerServiceAccountMutationResult{}, errRequest
	}
	d.Adds++
	return d.change(request.Computer, request.Accounts[0], true), nil
}

// RemoveComputerServiceAccounts removes exactly one configured direct link without deleting either endpoint.
func (d *Directory) RemoveComputerServiceAccounts(_ context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(request.Accounts) != 1 || !request.Permissive {
		return sdkaccounts.ComputerServiceAccountMutationResult{}, errRequest
	}
	d.Removes++
	return d.change(request.Computer, request.Accounts[0], false), nil
}

func (d *Directory) change(computer, account shared.Identity, present bool) sdkaccounts.ComputerServiceAccountMutationResult {
	if d.Links[computer.String()] == nil {
		d.Links[computer.String()] = map[string]bool{}
	}
	if present {
		d.Links[computer.String()][account.String()] = true
	} else {
		delete(d.Links[computer.String()], account.String())
	}
	guid, _ := shared.ParseGUID(computer.String())
	return sdkaccounts.ComputerServiceAccountMutationResult{
		Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid},
		Accounts: []shared.IdentityResult{{Object: objectPointer(d.Objects[account.String()]), Complete: true}},
	}
}

// SetLink applies independent unit-fixture drift without invoking the provider's mutation callbacks.
func (d *Directory) SetLink(computer, account string, present bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	computerIdentity, _ := shared.IdentityByGUID(computer)
	accountIdentity, _ := shared.IdentityByGUID(account)
	d.change(computerIdentity, accountIdentity, present)
}

// HasLink independently checks the synthetic direct link after Terraform operations.
func (d *Directory) HasLink(computer, account string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Links[computer][account]
}

// objectPointer copies a synthetic account observation for an acknowledged mutation.
func objectPointer(object shared.Object) *shared.Object { return &object }
