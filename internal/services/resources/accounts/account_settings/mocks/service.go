// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks provides explicit callbacks at the public SDK account-settings boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Service client.AccountSettings
}

// AccountSettingsService exposes exactly the supplied callbacks on this synthetic qualified session.
func (s *Session) AccountSettingsService() client.AccountSettings { return s.Service }

// Service rejects omitted operations and has no account creation/deletion or password APIs.
type Service struct {
	ReadControl     func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountControlResult, error)
	UpdateControl   func(context.Context, accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error)
	ReadExpiration  func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error)
	SetExpiration   func(context.Context, accounts.SetAccountExpirationRequest) (accounts.AccountExpirationMutationResult, error)
	ClearExpiration func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountExpirationMutationResult, error)
}

// AccountControl forwards an explicit typed read without inventing a result for omitted callbacks.
func (s *Service) AccountControl(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
	if s.ReadControl == nil {
		return accounts.AccountControlResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.ReadControl(ctx, target, options)
}

// UpdateAccountControl forwards one typed atomic patch.
func (s *Service) UpdateAccountControl(ctx context.Context, request accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
	if s.UpdateControl == nil {
		return accounts.AccountControlMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.UpdateControl(ctx, request)
}

// AccountExpiration forwards a GUID-pinned read.
func (s *Service) AccountExpiration(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error) {
	if s.ReadExpiration == nil {
		return accounts.AccountExpirationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.ReadExpiration(ctx, target, options)
}

// SetAccountExpiration forwards one finite expiration request.
func (s *Service) SetAccountExpiration(ctx context.Context, request accounts.SetAccountExpirationRequest) (accounts.AccountExpirationMutationResult, error) {
	if s.SetExpiration == nil {
		return accounts.AccountExpirationMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.SetExpiration(ctx, request)
}

// ClearAccountExpiration forwards only an explicit never-expire operation.
func (s *Service) ClearAccountExpiration(ctx context.Context, target shared.Identity, options shared.ObjectMutationOptions) (accounts.AccountExpirationMutationResult, error) {
	if s.ClearExpiration == nil {
		return accounts.AccountExpirationMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.ClearExpiration(ctx, target, options)
}
