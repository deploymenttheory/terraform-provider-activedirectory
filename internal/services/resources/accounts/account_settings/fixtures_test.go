// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"sync"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/accounts/account_settings/mocks"
)

const secondGUID = "a3e9767a-fb25-4d2c-849a-14d672c61005"

type unitAccount struct {
	guid       string
	dn         string
	flags      map[string]bool
	expiration shared.FileTime
	lastSet    shared.FileTime
	exists     bool
}

type unitDirectory struct {
	mu       sync.Mutex
	accounts map[string]*unitAccount
	calls    map[string]int
	patches  []map[string]bool
	hidden   map[string]bool
}

func newUnitDirectory(t *testing.T) *unitDirectory {
	t.Helper()
	payload, err := os.ReadFile("tests/responses/account.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		GUID       string          `json:"guid"`
		DN         string          `json:"distinguishedName"`
		Enabled    bool            `json:"enabled"`
		Expiration shared.FileTime `json:"accountExpires"`
		LastSet    shared.FileTime `json:"passwordLastSet"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	directory := &unitDirectory{accounts: map[string]*unitAccount{}, calls: map[string]int{}, hidden: map[string]bool{}}
	for _, id := range []string{fixture.GUID, secondGUID} {
		directory.accounts[id] = &unitAccount{guid: id, dn: fixture.DN, expiration: fixture.Expiration, lastSet: fixture.LastSet, exists: true, flags: map[string]bool{}}
		if id == secondGUID {
			directory.accounts[id].dn = "CN=Second,OU=owned,DC=unit,DC=invalid"
		}
		for name := range flagFieldNames() {
			directory.accounts[id].flags[name] = false
		}
		directory.accounts[id].flags["enabled"] = fixture.Enabled
		directory.accounts[id].flags["home_directory_required"] = true
	}
	return directory
}

func flagFieldNames() map[string]bool {
	model, patch := AccountSettingsResourceModel{}, accounts.AccountControlPatch{}
	result := map[string]bool{}
	for _, field := range flagProperties(&model, &patch) {
		result[field.name] = true
	}
	return result
}

func (d *unitDirectory) resolve(target shared.Identity) *unitAccount {
	if target.Form() == shared.IdentityFormGUID {
		row := d.accounts[target.String()]
		if row != nil && row.exists {
			return row
		}
		return nil
	}
	for _, row := range d.accounts {
		if row.exists && row.dn == target.String() {
			return row
		}
	}
	return nil
}

func rawAccount(row *unitAccount) shared.Object {
	guid, _ := shared.ParseGUID(row.guid)
	return shared.Object{DistinguishedName: row.dn, Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "distinguishedname": {[]byte(row.dn)}, "objectclass": {[]byte("top"), []byte("person"), []byte("organizationalPerson"), []byte("user")},
	}}
}

func (d *unitDirectory) control(row *unitAccount) accounts.AccountControlState {
	state := accounts.AccountControlState{DistinguishedName: row.dn, Raw: rawAccount(row)}
	fields := map[string]**bool{
		"enabled": &state.Enabled, "account_not_delegated": &state.AccountNotDelegated, "allow_reversible_password_encryption": &state.AllowReversiblePasswordEncryption,
		"does_not_require_preauth": &state.DoesNotRequirePreAuth, "home_directory_required": &state.HomeDirectoryRequired, "mns_logon_account": &state.MNSLogonAccount,
		"password_never_expires": &state.PasswordNeverExpires, "password_not_required": &state.PasswordNotRequired, "smartcard_logon_required": &state.SmartcardLogonRequired,
		"trusted_for_delegation": &state.TrustedForDelegation, "trusted_to_auth_for_delegation": &state.TrustedToAuthForDelegation, "use_des_key_only": &state.UseDESKeyOnly,
	}
	for name, destination := range fields {
		if !d.hidden[name] {
			value := row.flags[name]
			*destination = &value
		}
	}
	if !d.hidden["change_password_at_logon"] {
		stamp := row.lastSet
		state.PasswordLastSet = &stamp
	}
	locked, expired := false, false
	state.LockedOut, state.PasswordExpired = &locked, &expired
	return state
}

func (d *unitDirectory) expiration(row *unitAccount) accounts.AccountExpiration {
	result := accounts.AccountExpiration{DistinguishedName: row.dn, Raw: rawAccount(row)}
	if d.hidden["expiration"] {
		return result
	}
	value := row.expiration
	result.Value = &value
	never := value == 0 || value == math.MaxInt64
	result.NeverExpires = &never
	if !never {
		at, _ := value.Time()
		result.ExpiresAt = &at
	}
	return result
}

func (d *unitDirectory) callbacks() *featuremocks.Service {
	service := &featuremocks.Service{}
	service.ReadControl = func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.calls["read_control"]++
		row := d.resolve(target)
		if row == nil {
			return accounts.AccountControlResult{}, shared.ErrDirectoryObjectNotFound
		}
		control := d.control(row)
		return accounts.AccountControlResult{Account: &control, Observation: shared.IdentityResult{Object: &control.Raw, Complete: true}}, nil
	}
	service.ReadExpiration = func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.calls["read_expiration"]++
		row := d.resolve(target)
		if row == nil {
			return accounts.AccountExpirationResult{}, shared.ErrDirectoryObjectNotFound
		}
		value := d.expiration(row)
		return accounts.AccountExpirationResult{Expiration: &value, Observation: shared.IdentityResult{Object: &value.Raw, Complete: true}}, nil
	}
	service.UpdateControl = func(_ context.Context, request accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.calls["flags"]++
		row := d.resolve(request.Identity)
		if row == nil {
			return accounts.AccountControlMutationResult{}, shared.ErrDirectoryObjectNotFound
		}
		model := AccountSettingsResourceModel{}
		patch := request.Patch
		changed := map[string]bool{}
		for _, field := range flagProperties(&model, &patch) {
			if *field.target == nil {
				continue
			}
			value := **field.target
			changed[field.name] = value
			row.flags[field.name] = value
			if field.name == "change_password_at_logon" {
				row.lastSet = 133000000000000000
				if value {
					row.lastSet = 0
				}
			}
		}
		d.patches = append(d.patches, changed)
		control := d.control(row)
		guid, _ := shared.ParseGUID(row.guid)
		return accounts.AccountControlMutationResult{
			Account:  &control,
			Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, After: &control.Raw, Verified: true},
		}, nil
	}
	service.SetExpiration = func(_ context.Context, request accounts.SetAccountExpirationRequest) (accounts.AccountExpirationMutationResult, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.calls["set_expiration"]++
		row := d.resolve(request.Identity)
		if row == nil {
			return accounts.AccountExpirationMutationResult{}, shared.ErrDirectoryObjectNotFound
		}
		row.expiration, _ = shared.FileTimeFromTime(request.At)
		value := d.expiration(row)
		guid, _ := shared.ParseGUID(row.guid)
		return accounts.AccountExpirationMutationResult{
			Expiration: &value,
			Mutation:   shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, After: &value.Raw, Verified: true},
		}, nil
	}
	service.ClearExpiration = func(_ context.Context, target shared.Identity, _ shared.ObjectMutationOptions) (accounts.AccountExpirationMutationResult, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.calls["clear_expiration"]++
		row := d.resolve(target)
		if row == nil {
			return accounts.AccountExpirationMutationResult{}, shared.ErrDirectoryObjectNotFound
		}
		row.expiration = math.MaxInt64
		value := d.expiration(row)
		guid, _ := shared.ParseGUID(row.guid)
		return accounts.AccountExpirationMutationResult{
			Expiration: &value,
			Mutation:   shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, After: &value.Raw, Verified: true},
		}, nil
	}
	return service
}
