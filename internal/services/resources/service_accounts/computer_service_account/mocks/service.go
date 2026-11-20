// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies isolated direct-link callbacks at the public SDK boundary.
package mocks

import (
	"context"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	ComputerServiceAccountLinks client.ComputerServiceAccountLinks
}

// ComputerServiceAccountService exposes the explicitly supplied direct-link operations.
func (s *Session) ComputerServiceAccountService() client.ComputerServiceAccountLinks {
	return s.ComputerServiceAccountLinks
}

// Service refuses omitted callbacks instead of contacting a directory or inventing a successful mutation.
type Service struct {
	Resolve func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Assert  func(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	Links   func(context.Context, shared.Identity, sdkaccounts.ComputerServiceAccountsOptions) (sdkaccounts.ComputerServiceAccountsResult, error)
	Add     func(context.Context, sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error)
	Remove  func(context.Context, sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error)
}

// ResolveIdentity forwards only the explicitly configured callback.
func (s *Service) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	if s.Resolve == nil {
		return shared.IdentityResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Resolve(ctx, target, options)
}

// Search forwards only the explicitly configured callback.
func (s *Service) Search(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	if s.Assert == nil {
		return shared.SearchResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Assert(ctx, options)
}

// ComputerServiceAccounts forwards only the explicitly configured callback.
func (s *Service) ComputerServiceAccounts(
	ctx context.Context,
	target shared.Identity,
	options sdkaccounts.ComputerServiceAccountsOptions,
) (sdkaccounts.ComputerServiceAccountsResult, error) {
	if s.Links == nil {
		return sdkaccounts.ComputerServiceAccountsResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Links(ctx, target, options)
}

// AddComputerServiceAccounts forwards only the explicitly configured callback.
func (s *Service) AddComputerServiceAccounts(ctx context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
	if s.Add == nil {
		return sdkaccounts.ComputerServiceAccountMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Add(ctx, request)
}

// RemoveComputerServiceAccounts forwards only the explicitly configured callback.
func (s *Service) RemoveComputerServiceAccounts(ctx context.Context, request sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error) {
	if s.Remove == nil {
		return sdkaccounts.ComputerServiceAccountMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Remove(ctx, request)
}
