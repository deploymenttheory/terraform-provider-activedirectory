// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// AccountSettings exposes typed reads and scoped updates for an existing account; it cannot create or delete accounts.
type AccountSettings interface {
	AccountControl(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountControlResult, error)
	UpdateAccountControl(context.Context, accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error)
	AccountExpiration(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error)
	SetAccountExpiration(context.Context, accounts.SetAccountExpirationRequest) (accounts.AccountExpirationMutationResult, error)
	ClearAccountExpiration(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountExpirationMutationResult, error)
}

var ErrAccountSettings = errors.New("configured SDK session does not expose account settings")

// AccountSettings returns adopted-account services sharing the bundle's authenticated session and lifetime.
func (b *Bundle) AccountSettings() (AccountSettings, error) {
	if b == nil || b.session == nil {
		return nil, ErrAccountSettings
	}
	bound, ok := b.session.(interface{ AccountSettingsService() AccountSettings })
	if !ok {
		return nil, ErrAccountSettings
	}
	service := bound.AccountSettingsService()
	if service == nil {
		return nil, ErrAccountSettings
	}
	return service, nil
}

// AccountSettingsService binds public SDK account settings to the existing protected connection.
func (s *sdkSession) AccountSettingsService() AccountSettings { return s.Accounts }
