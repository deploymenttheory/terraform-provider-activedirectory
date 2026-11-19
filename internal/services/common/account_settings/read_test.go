// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

type readFixture struct {
	client.AccountSettings
	control                       accounts.AccountControlResult
	expiration                    accounts.AccountExpirationResult
	controlErr, errorAfterResolve error
	expirationTarget              shared.Identity
	controlCalls, expirationCalls int
}

func (f *readFixture) AccountControl(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
	f.controlCalls++
	return f.control, f.controlErr
}

func (f *readFixture) AccountExpiration(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error) {
	f.expirationCalls++
	f.expirationTarget = target
	return f.expiration, f.errorAfterResolve
}

// TestReadPinsSubsequentObservationToResolvedGUID prevents DN reuse between reads from silently adopting another account.
func TestReadPinsSubsequentObservationToResolvedGUID(t *testing.T) {
	t.Parallel()
	raw := observation(t, "dc93656c-f745-4926-9bbb-31e7bbecb68b")
	service := &readFixture{
		control:    accounts.AccountControlResult{Account: &accounts.AccountControlState{Raw: raw, DistinguishedName: raw.DistinguishedName}},
		expiration: accounts.AccountExpirationResult{Expiration: &accounts.AccountExpiration{Raw: raw}},
	}
	target, err := shared.IdentityByDN(raw.DistinguishedName)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Read(context.Background(), service, target)
	if err != nil || result.Values == nil || service.controlCalls != 1 || service.expirationCalls != 1 || service.expirationTarget.Form() != shared.IdentityFormGUID ||
		service.expirationTarget.String() != "dc93656c-f745-4926-9bbb-31e7bbecb68b" {
		t.Fatal("the later lookup reused a mutable name or did not emit the qualified snapshot")
	}
	service.expiration.Expiration.Raw = observation(t, "a3e9767a-fb25-4d2c-849a-14d672c61005")
	result, err = Read(context.Background(), service, target)
	if !errors.Is(err, ErrObservation) || result.Values != nil || result.ConfirmedAbsent {
		t.Fatal("a contradictory later observation emitted state or an absence claim")
	}
}

// TestReadStopsAfterFailedResolution never adds a second lookup or retry after a failed or mismatched identity observation.
func TestReadStopsAfterFailedResolution(t *testing.T) {
	t.Parallel()
	raw := observation(t, "dc93656c-f745-4926-9bbb-31e7bbecb68b")
	target, _ := shared.IdentityByGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	for _, service := range []*readFixture{
		{controlErr: ErrObservation},
		{},
		{control: accounts.AccountControlResult{Account: &accounts.AccountControlState{Raw: raw, DistinguishedName: raw.DistinguishedName}}},
	} {
		result, err := Read(context.Background(), service, target)
		if err == nil || result.Values != nil || result.ConfirmedAbsent || service.controlCalls != 1 || service.expirationCalls != 0 {
			t.Fatal("a failed or wrong identity triggered a later lookup or fabricated absence")
		}
	}
}

// TestReadAbsenceRejectsPositiveOrMixedObservations retains adopted settings when any returned account or error contradicts absence.
func TestReadAbsenceRejectsPositiveOrMixedObservations(t *testing.T) {
	t.Parallel()
	target, _ := shared.IdentityByGUID("dc93656c-f745-4926-9bbb-31e7bbecb68b")
	for _, row := range []struct {
		account *accounts.AccountControlState
		err     error
		absent  bool
	}{
		{nil, shared.ErrDirectoryObjectNotFound, true},
		{nil, errors.Join(shared.ErrDirectoryObjectNotFound, shared.ErrObjectWritePermission), false},
		{&accounts.AccountControlState{}, shared.ErrDirectoryObjectNotFound, false},
	} {
		service := &readFixture{control: accounts.AccountControlResult{Account: row.account}, controlErr: row.err}
		result, err := Read(context.Background(), service, target)
		if err == nil || result.ConfirmedAbsent != row.absent || service.expirationCalls != 0 {
			t.Fatal("contradictory account data or errors authorized state removal")
		}
	}
}
