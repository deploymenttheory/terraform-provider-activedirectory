// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// ComputerServiceAccountLinks exposes direct links and individual association changes on the qualified provider session.
// Principal resolution supplies stable GUIDs without installing accounts or retrieving managed passwords.
type ComputerServiceAccountLinks interface {
	ResolveIdentity(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Search(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	ComputerServiceAccounts(context.Context, shared.Identity, sdkaccounts.ComputerServiceAccountsOptions) (sdkaccounts.ComputerServiceAccountsResult, error)
	AddComputerServiceAccounts(context.Context, sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error)
	RemoveComputerServiceAccounts(context.Context, sdkaccounts.ComputerServiceAccountRequest) (sdkaccounts.ComputerServiceAccountMutationResult, error)
}

var ErrComputerServiceAccountLinks = errors.New("configured SDK session does not expose direct computer association operations")

// ComputerServiceAccountLinks returns direct-link operations without creating a connection or changing authentication.
func (b *Bundle) ComputerServiceAccountLinks() (ComputerServiceAccountLinks, error) {
	if b == nil || b.session == nil {
		return nil, ErrComputerServiceAccountLinks
	}
	getter, ok := b.session.(interface {
		ComputerServiceAccountService() ComputerServiceAccountLinks
	})
	if !ok {
		return nil, ErrComputerServiceAccountLinks
	}
	service := getter.ComputerServiceAccountService()
	if service == nil {
		return nil, ErrComputerServiceAccountLinks
	}
	return service, nil
}

// ComputerServiceAccountService combines public SDK association and identity operations on the existing session.
func (s *sdkSession) ComputerServiceAccountService() ComputerServiceAccountLinks {
	return &sdkComputerServiceAccountLinks{Service: s.ServiceAccounts, session: s}
}

type sdkComputerServiceAccountLinks struct {
	*sdkaccounts.Service
	session *sdkSession
}

// ResolveIdentity reads a direct account's own object identity without following association links.
func (s *sdkComputerServiceAccountLinks) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	result, err := s.session.Directory.ResolveIdentity(ctx, target, options)
	if err != nil {
		return result, fmt.Errorf("resolve direct account identity: %w", err)
	}
	return result, nil
}

// Search delegates bounded direct-link assertions to the directory through the public SDK.
func (s *sdkComputerServiceAccountLinks) Search(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	result, err := s.session.Directory.Search(ctx, options)
	if err != nil {
		return result, fmt.Errorf("assert direct computer association: %w", err)
	}
	return result, nil
}
