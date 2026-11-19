// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Accounts exposes account operations over the bundle's existing authenticated SDK session.
type Accounts interface {
	UnlockAccount(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error)
}

var ErrAccounts = errors.New("configured SDK session does not expose account operations")

// Accounts returns account services sharing this bundle's protection, selected server and lifetime.
func (b *Bundle) Accounts() (Accounts, error) {
	if b == nil || b.session == nil {
		return nil, ErrAccounts
	}
	bound, ok := b.session.(interface{ AccountService() Accounts })
	if !ok {
		return nil, ErrAccounts
	}
	service := bound.AccountService()
	if service == nil {
		return nil, ErrAccounts
	}
	return service, nil
}

// AccountService binds typed SDK operations to this session rather than opening another connection.
func (s *sdkSession) AccountService() Accounts { return s.Accounts }

// ServerName returns the server's observed DNS identity; a missing observation is never replaced with a guessed name.
func (b *Bundle) ServerName() (string, bool) {
	if b == nil || b.root.DNSHostName == nil || *b.root.DNSHostName == "" {
		return "", false
	}
	return *b.root.DNSHostName, true
}
