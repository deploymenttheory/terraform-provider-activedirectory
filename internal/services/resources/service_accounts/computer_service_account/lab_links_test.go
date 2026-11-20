// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"strings"
	"testing"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// setLink introduces one known fixture link or independent drift only after rechecking both endpoint boundaries.
func (l *computerServiceAccountLab) setLink(t *testing.T, computer, account labSubject, present bool) {
	t.Helper()
	if computer.Kind != "computer" {
		t.Fatal("association fixture owner is not a computer")
	}
	for _, subject := range []labSubject{computer, account} {
		if _, err := l.ownedObject(t.Context(), subject); err != nil {
			t.Fatal("fixture ownership did not verify before linked-value mutation")
		}
	}
	var result sdkaccounts.ComputerServiceAccountMutationResult
	var err error
	if present {
		result, err = l.observer.ServiceAccounts.AddComputerServiceAccounts(t.Context(), sdkaccounts.ComputerServiceAccountRequest{
			Computer: l.identity(computer), Accounts: []shared.Identity{l.identity(account)},
		})
	} else {
		result, err = l.observer.ServiceAccounts.RemoveComputerServiceAccounts(t.Context(), sdkaccounts.ComputerServiceAccountRequest{
			Computer: l.identity(computer), Accounts: []shared.Identity{l.identity(account)},
		})
	}
	if err != nil || result.Mutation.State != shared.ObjectWriteAccepted || result.Mutation.TargetGUID == nil || result.Mutation.TargetGUID.String() != computer.GUID {
		t.Fatal("independent direct-link mutation was not acknowledged; no automatic retry is authorized")
	}
	if err := l.assertLink(computer, account, present); err != nil {
		t.Fatal("independent direct-link fixture readback did not verify")
	}
}

// assertLink observes direct values through the independent administrative fixture session, separate from provider assertion mapping.
func (l *computerServiceAccountLab) assertLink(computer, account labSubject, present bool) error {
	if _, err := l.ownedObject(context.Background(), computer); err != nil {
		return errLabOwnership
	}
	object, err := l.ownedObject(context.Background(), account)
	if err != nil {
		return errLabOwnership
	}
	observed, err := l.observer.ServiceAccounts.ComputerServiceAccounts(
		context.Background(),
		l.identity(computer),
		sdkaccounts.ComputerServiceAccountsOptions{MaxValues: 20, MaxRanges: 5, MaxBytes: 1 << 20},
	)
	if err != nil || !observed.Complete || observed.Version == nil || observed.Version.GUID.String() != computer.GUID {
		return errLabOwnership
	}
	found := false
	for _, link := range observed.Links {
		if strings.EqualFold(link.DistinguishedName, object.DistinguishedName) {
			found = true
		}
	}
	if found != present {
		return errLabOwnership
	}
	return nil
}
