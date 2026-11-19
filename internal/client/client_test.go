// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package client_test

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

type sessionFixture struct {
	protection activedirectory.SessionProtection
	root       shared.RootDSE
	readErr    error
	closed     int
	reads      int
}

func (s *sessionFixture) RootDSE(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
	s.reads++
	return s.root, s.readErr
}

func (s *sessionFixture) Protection(context.Context) (activedirectory.SessionProtection, error) {
	return s.protection, nil
}

func (s *sessionFixture) Close() error { s.closed++; return nil }

func simpleTLSConfig() config.Config {
	return config.Config{
		Backend: config.BackendPortable, Transport: config.TransportLDAPS, Authentication: config.AuthenticationSimple,
		Endpoint: config.Endpoint{Host: "dc.example.test"}, Credentials: &config.Credentials{Username: "test@example.test", Password: []byte("test-only")},
	}
}

func protectedSession(options config.Config) *sessionFixture {
	return &sessionFixture{
		protection: activedirectory.SessionProtection{
			Backend: options.Backend, Transport: options.Transport, Authentication: options.Authentication, Complete: true,
			TLSVerified: true, TLSVersion: tls.VersionTLS13, TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256,
			MutualAuthentication: true, ChannelBindingSent: true,
		},
		root: shared.RootDSE{SupportedCapabilities: []string{"1.2.840.113556.1.4.800"}},
	}
}

func TestOpenRejectsIncompleteOrDowngradedProtection(t *testing.T) {
	t.Parallel()
	tests := map[string]func(*sessionFixture){
		"incomplete":           func(s *sessionFixture) { s.protection.Complete = false },
		"wrong backend":        func(s *sessionFixture) { s.protection.Backend = config.BackendNative },
		"wrong authentication": func(s *sessionFixture) { s.protection.Authentication = config.AuthenticationNegotiate },
		"wrong transport":      func(s *sessionFixture) { s.protection.Transport = config.TransportLDAP },
		"unverified TLS":       func(s *sessionFixture) { s.protection.TLSVerified = false },
		"obsolete TLS":         func(s *sessionFixture) { s.protection.TLSVersion = tls.VersionTLS11 },
		"weak encryption":      func(s *sessionFixture) { s.protection.CipherBits = 64 },
	}
	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			options := simpleTLSConfig()
			session := protectedSession(options)
			modify(session)
			bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) { return session, nil })
			if !errors.Is(err, client.ErrProtection) || bundle != nil || session.closed != 1 || session.reads != 0 {
				t.Fatalf("unprotected session was not retired before reading: bundle=%v err=%v closes=%d reads=%d", bundle, err, session.closed, session.reads)
			}
		})
	}
}

func TestOpenQualifiesKerberosProtection(t *testing.T) {
	t.Parallel()
	for _, transport := range []config.Transport{config.TransportLDAPS, config.TransportStartTLS, config.TransportLDAP} {
		for _, mode := range []config.SASLProtection{config.SASLProtectionDefault, config.SASLProtectionSign, config.SASLProtectionSeal} {
			if transport != config.TransportLDAP && mode != config.SASLProtectionDefault {
				continue
			}
			options := simpleTLSConfig()
			options.Transport, options.Authentication, options.Credentials = transport, config.AuthenticationKerberos, nil
			options.SASLProtection = mode
			options.Kerberos = &config.KerberosCredentials{
				Source: config.KerberosPassword, Username: "test", Password: []byte("test-only"), Realm: "EXAMPLE.TEST", SPN: "ldap/dc.example.test",
			}
			session := protectedSession(options)
			if transport == config.TransportLDAP {
				session.protection.TLSVerified = false
				session.protection.ChannelBindingSent = false
				session.protection.SASLProtection = mode
				if mode == config.SASLProtectionDefault {
					session.protection.SASLProtection = config.SASLProtectionSeal
				}
				if session.protection.SASLProtection == config.SASLProtectionSeal {
					session.protection.SASLCipherBits = 256
				}
			}
			dial := func(context.Context, config.Config) (client.Session, error) { return session, nil }
			bundle, err := client.OpenWithDial(context.Background(), options, dial)
			if err != nil {
				t.Fatalf("qualified Kerberos session rejected: %v", err)
			}
			if err := bundle.Close(); err != nil {
				t.Fatal(err)
			}
			if err := bundle.Close(); err != nil {
				t.Fatal(err)
			}
			if session.closed != 1 {
				t.Fatalf("expected idempotent close, got %d", session.closed)
			}
			session.protection.MutualAuthentication = false
			if _, err := client.OpenWithDial(context.Background(), options, dial); !errors.Is(err, client.ErrProtection) {
				t.Fatalf("missing mutual authentication accepted: %v", err)
			}
		}
	}
}

func TestOpenRejectsMissingChannelBinding(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	options.Authentication, options.Credentials = config.AuthenticationKerberos, nil
	options.Kerberos = &config.KerberosCredentials{
		Source: config.KerberosPassword, Username: "test", Password: []byte("test-only"), Realm: "EXAMPLE.TEST", SPN: "ldap/dc.example.test",
	}
	session := protectedSession(options)
	session.protection.ChannelBindingSent = false
	_, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) { return session, nil })
	if !errors.Is(err, client.ErrProtection) || session.closed != 1 {
		t.Fatalf("missing TLS channel binding accepted: %v", err)
	}
}

func TestOpenRejectsPartialOrNonADDSCapabilities(t *testing.T) {
	t.Parallel()
	for _, partial := range []bool{false, true} {
		options := simpleTLSConfig()
		session := protectedSession(options)
		if partial {
			session.readErr = errors.New("partial RootDSE")
		} else {
			session.root.SupportedCapabilities = nil
		}
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) { return session, nil })
		if err == nil || bundle != nil || session.closed != 1 {
			t.Fatalf("unqualified directory accepted: %v", err)
		}
	}
}
