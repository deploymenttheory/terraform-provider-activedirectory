// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies explicit callbacks at the public SDK service account boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	ServiceAccounts client.ServiceAccounts
}

// ServiceAccountService exposes only the supplied service account callbacks on this synthetic authenticated session.
func (s *Session) ServiceAccountService() client.ServiceAccounts { return s.ServiceAccounts }

// Service rejects every omitted callback rather than inventing successful writes or contacting AD.
type Service struct {
	Read   func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (service_accounts.ServiceAccountResult, error)
	List   func(context.Context, shared.SearchOptions) (service_accounts.ServiceAccountsResult, error)
	Create func(context.Context, service_accounts.CreateServiceAccountRequest) (service_accounts.ServiceAccountCreationResult, error)
	Update func(context.Context, service_accounts.UpdateServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error)
	Delete func(context.Context, service_accounts.DeleteServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error)
	Rename func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move   func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

// Service account forwards an explicit lookup.
func (s *Service) ServiceAccount(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (service_accounts.ServiceAccountResult, error) {
	if s.Read == nil {
		return service_accounts.ServiceAccountResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, target, options)
}

// ServiceAccounts forwards an explicit bounded search.
func (s *Service) ServiceAccounts(ctx context.Context, options shared.SearchOptions) (service_accounts.ServiceAccountsResult, error) {
	if s.List == nil {
		return service_accounts.ServiceAccountsResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateService account forwards a single staged creation request.
func (s *Service) CreateServiceAccount(ctx context.Context, request service_accounts.CreateServiceAccountRequest) (service_accounts.ServiceAccountCreationResult, error) {
	if s.Create == nil {
		return service_accounts.ServiceAccountCreationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateService account forwards only explicitly constructed changes.
func (s *Service) UpdateServiceAccount(ctx context.Context, request service_accounts.UpdateServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error) {
	if s.Update == nil {
		return service_accounts.ServiceAccountMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteService account forwards a leaf-only service account deletion request.
func (s *Service) DeleteServiceAccount(ctx context.Context, request service_accounts.DeleteServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error) {
	if s.Delete == nil {
		return service_accounts.ServiceAccountMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Delete(ctx, request)
}

// RenameObject forwards a same-GUID rename.
func (s *Service) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Rename == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Rename(ctx, request)
}

// MoveObject forwards an explicitly selected same-domain move.
func (s *Service) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	if s.Move == nil {
		return shared.ObjectMoveResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Move(ctx, request)
}
