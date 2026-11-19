// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"context"
	"crypto/tls"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

// Factory supplies an explicitly synthetic portable TLS fixture session for isolated computer feature factories.
func Factory(computers client.Computers) client.Factory {
	return sdkmocks.Factory(func() client.Session {
		return &Session{Computers: computers, Session: &sdkmocks.Session{
			RootDSEFunc: func(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
				partition := "DC=unit,DC=invalid"
				return shared.RootDSE{DefaultNamingContext: &partition, SupportedCapabilities: []string{"1.2.840.113556.1.4.800"}}, nil
			},
			ProtectionFunc: func(context.Context) (activedirectory.SessionProtection, error) {
				return activedirectory.SessionProtection{
					Backend: config.BackendPortable, Authentication: config.AuthenticationSimple,
					Transport: config.TransportLDAPS, Complete: true, TLSVerified: true, TLSVersion: tls.VersionTLS13,
					TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256,
				}, nil
			},
		}}
	})
}
