// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Session is the public SDK connection contract needed by provider configuration.
// Service-specific interfaces are added alongside the features that consume them.
type Session interface {
	RootDSE(context.Context, shared.RootDSEOptions) (shared.RootDSE, error)
	Protection(context.Context) (activedirectory.SessionProtection, error)
	Close() error
}

// Factory opens one configured client bundle. Tests inject factories without live credentials.
type Factory func(context.Context, config.Config) (*Bundle, error)

// DialFunc creates an SDK session using exactly the requested authentication and transport.
type DialFunc func(context.Context, config.Config) (Session, error)

// Bundle owns the authenticated session shared by resources, data sources, list resources and actions.
type Bundle struct {
	session    Session
	protection activedirectory.SessionProtection
	root       shared.RootDSE
	closeOnce  sync.Once
	closeErr   error
}

var (
	ErrProtection = errors.New("directory session did not establish the required protection")
	ErrSession    = errors.New("directory factory returned no session")
)

// Open constructs and qualifies one SDK session without changing directory objects.
func Open(ctx context.Context, options config.Config) (*Bundle, error) {
	return OpenWithDial(ctx, options, dialSDK)
}

// OpenWithDial permits isolated tests to supply a session implementation over public SDK contracts.
// Failed or incomplete qualification always closes the newly opened session.
func OpenWithDial(ctx context.Context, options config.Config, dial DialFunc) (*Bundle, error) {
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("validate directory client configuration: %w", err)
	}
	session, err := dial(ctx, options)
	if err != nil {
		if session != nil {
			err = errors.Join(err, session.Close())
		}
		return nil, fmt.Errorf("open directory session: %w", err)
	}
	if session == nil {
		return nil, ErrSession
	}
	protection, err := session.Protection(ctx)
	if err == nil {
		err = validateProtection(options, protection)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("verify directory session protection: %w", err), session.Close())
	}
	root, err := session.RootDSE(ctx, shared.RootDSEOptions{})
	if err == nil {
		err = root.Capabilities().RequireADDS("Terraform provider configuration")
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("observe directory capabilities: %w", err), session.Close())
	}
	return &Bundle{session: session, protection: protection, root: root}, nil
}

// Directory returns the configured session. Its SDK operations serialize access and honor caller contexts.
func (b *Bundle) Directory() Session { return b.session }

// Protection reports measured security properties rather than inferring them from requested options.
func (b *Bundle) Protection() activedirectory.SessionProtection { return b.protection }

// Capabilities returns an independent copy of the observed AD DS capability projection.
func (b *Bundle) Capabilities() shared.DirectoryCapabilities { return b.root.Capabilities() }

// Close releases the shared session. SDK sessions provide concurrent, idempotent closure.
func (b *Bundle) Close() error {
	if b == nil || b.session == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		if err := b.session.Close(); err != nil {
			b.closeErr = fmt.Errorf("close directory session: %w", err)
		}
	})
	return b.closeErr
}

func validateProtection(options config.Config, observed activedirectory.SessionProtection) error {
	if !observed.Complete || observed.Backend != options.Backend || observed.Authentication != options.Authentication || observed.Transport != options.Transport {
		return ErrProtection
	}
	if options.Transport != config.TransportLDAP && (!observed.TLSVerified || observed.TLSVersion < tls.VersionTLS12 || observed.TLSCipherSuite == 0 || observed.CipherBits < 128) {
		return ErrProtection
	}
	if options.Authentication == config.AuthenticationKerberos {
		if !observed.MutualAuthentication || (options.Transport != config.TransportLDAP && !observed.ChannelBindingSent) {
			return ErrProtection
		}
	}
	if options.Transport == config.TransportLDAP {
		requested := options.SASLProtection
		if requested == config.SASLProtectionDefault {
			requested = config.SASLProtectionSeal
		}
		if observed.SASLProtection != requested || (requested == config.SASLProtectionSeal && observed.SASLCipherBits < 128) {
			return ErrProtection
		}
	}
	return nil
}

type sdkSession struct{ *activedirectory.Client }

func dialSDK(ctx context.Context, options config.Config) (Session, error) {
	sdk, err := activedirectory.Dial(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("dial Active Directory SDK: %w", err)
	}
	return &sdkSession{Client: sdk}, nil
}

func (s *sdkSession) RootDSE(ctx context.Context, options shared.RootDSEOptions) (shared.RootDSE, error) {
	root, err := s.Directory.RootDSE(ctx, options)
	if err != nil {
		return root, fmt.Errorf("read directory RootDSE: %w", err)
	}
	return root, nil
}
