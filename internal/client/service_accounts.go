// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// ServiceAccounts binds service account lifecycle and location changes to the provider's existing qualified SDK session.
type ServiceAccounts interface {
	ServiceAccount(context.Context, shared.Identity, shared.ResolveIdentityOptions) (service_accounts.ServiceAccountResult, error)
	ServiceAccounts(context.Context, shared.SearchOptions) (service_accounts.ServiceAccountsResult, error)
	CreateServiceAccount(context.Context, service_accounts.CreateServiceAccountRequest) (service_accounts.ServiceAccountCreationResult, error)
	UpdateServiceAccount(context.Context, service_accounts.UpdateServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error)
	DeleteServiceAccount(context.Context, service_accounts.DeleteServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

var ErrServiceAccounts = errors.New("configured SDK session does not expose service account operations")

// ServiceAccounts returns services with the same authenticated connection and lifetime as provider configuration.
func (b *Bundle) ServiceAccounts() (ServiceAccounts, error) {
	if b == nil || b.session == nil {
		return nil, ErrServiceAccounts
	}
	getter, ok := b.session.(interface{ ServiceAccountService() ServiceAccounts })
	if !ok {
		return nil, ErrServiceAccounts
	}
	service := getter.ServiceAccountService()
	if service == nil {
		return nil, ErrServiceAccounts
	}
	return service, nil
}

// ServiceAccountService composes public SDK operations without opening another session or retrieving managed passwords.
func (s *sdkSession) ServiceAccountService() ServiceAccounts {
	return &sdkServiceAccounts{Service: s.ServiceAccounts, session: s}
}

type sdkServiceAccounts struct {
	*service_accounts.Service
	session *sdkSession
}

// RenameObject changes the same service account's name through the generic SDK operation.
func (s *sdkServiceAccounts) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	result, err := s.session.Directory.RenameObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("rename service account: %w", err)
	}
	return result, nil
}

// MoveObject changes the service account's explicit location without changing its GUID or directory associations.
func (s *sdkServiceAccounts) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	result, err := s.session.Directory.MoveObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("move service account: %w", err)
	}
	return result, nil
}
