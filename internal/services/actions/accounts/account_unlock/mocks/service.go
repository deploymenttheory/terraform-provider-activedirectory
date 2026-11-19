// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks replaces only public account SDK operations for feature-local tests.
package mocks

import (
	"context"
	"crypto/tls"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

// Service refuses an omitted unlock callback instead of fabricating a successful write.
type Service struct {
	Unlock func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error)
}

// UnlockAccount forwards the selected identity and explicit options without retries.
func (s *Service) UnlockAccount(ctx context.Context, target shared.Identity, options shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
	if s.Unlock == nil {
		return accounts.AccountControlMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Unlock(ctx, target, options)
}

type session struct {
	*sdkmocks.Session
	accounts client.Accounts
}

// AccountService exposes the explicitly injected feature service over a fresh mock session.
func (s *session) AccountService() client.Accounts { return s.accounts }

// Factory qualifies synthetic TLS/AD DS observations without using live credentials or directory I/O.
func Factory(service client.Accounts) client.Factory {
	return sdkmocks.Factory(func() client.Session {
		return &session{accounts: service, Session: &sdkmocks.Session{
			RootDSEFunc: func(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
				partition, server := "DC=unit,DC=invalid", "dc.unit.invalid"
				return shared.RootDSE{DefaultNamingContext: &partition, DNSHostName: &server, SupportedCapabilities: []string{shared.CapabilityActiveDirectory}}, nil
			},
			ProtectionFunc: func(context.Context) (activedirectory.SessionProtection, error) {
				return activedirectory.SessionProtection{
					Backend: config.BackendPortable, Authentication: config.AuthenticationSimple, Transport: config.TransportLDAPS,
					Complete: true, TLSVerified: true, TLSVersion: tls.VersionTLS13, TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256,
				}, nil
			},
		}}
	})
}
