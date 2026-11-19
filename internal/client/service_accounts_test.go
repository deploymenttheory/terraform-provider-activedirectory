// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

type serviceAccountSession struct {
	*sessionFixture
	service client.ServiceAccounts
}

func (s *serviceAccountSession) ServiceAccountService() client.ServiceAccounts { return s.service }

type serviceAccountContract = client.ServiceAccounts

type serviceAccountOperations struct{ serviceAccountContract }

// TestServiceAccountsShareQualifiedSession verifies lookup neither substitutes the service nor opens another connection.
func TestServiceAccountsShareQualifiedSession(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	fixture := protectedSession(options)
	operations := &serviceAccountOperations{}
	session := &serviceAccountSession{sessionFixture: fixture, service: operations}
	dials := 0
	bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
		dials++
		return session, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		service, err := bundle.ServiceAccounts()
		if err != nil || service != operations {
			t.Fatal("managed-account operations did not retain the qualified service")
		}
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
		t.Fatal("managed-account surfaces did not share provider qualification and closure")
	}
}

// TestServiceAccountsRequireQualifiedService refuses absent capabilities without opening an additional session.
func TestServiceAccountsRequireQualifiedService(t *testing.T) {
	t.Parallel()
	var empty *client.Bundle
	if service, err := empty.ServiceAccounts(); service != nil || !errors.Is(err, client.ErrServiceAccounts) {
		t.Fatal("nil bundle exposed service account operations")
	}
	for _, withGetter := range []bool{false, true} {
		options := simpleTLSConfig()
		fixture := protectedSession(options)
		var session client.Session = fixture
		if withGetter {
			session = &serviceAccountSession{sessionFixture: fixture}
		}
		dials := 0
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
			dials++
			return session, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if service, err := bundle.ServiceAccounts(); service != nil || !errors.Is(err, client.ErrServiceAccounts) {
			t.Fatal("unavailable service-account service was not rejected")
		}
		if err := bundle.Close(); err != nil {
			t.Fatal(err)
		}
		if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
			t.Fatal("service lookup changed the qualified connection or its lifetime")
		}
	}
}
