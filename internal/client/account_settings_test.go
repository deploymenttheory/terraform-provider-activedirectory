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

type accountSettingsSession struct {
	*sessionFixture
	service client.AccountSettings
}

func (s *accountSettingsSession) AccountSettingsService() client.AccountSettings { return s.service }

type accountSettingsOperations struct{ client.AccountSettings }

// TestAccountSettingsShareQualifiedSession verifies lookup neither substitutes the service nor opens another connection.
func TestAccountSettingsShareQualifiedSession(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	fixture := protectedSession(options)
	operations := &accountSettingsOperations{}
	session := &accountSettingsSession{sessionFixture: fixture, service: operations}
	dials := 0
	bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
		dials++
		return session, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		service, err := bundle.AccountSettings()
		if err != nil || service != operations {
			t.Fatal("account-settings operations did not retain the qualified service")
		}
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
		t.Fatal("adopted-account surfaces did not share provider qualification and closure")
	}
}

// TestAccountSettingsRequireQualifiedService refuses absent capabilities without opening an additional session.
func TestAccountSettingsRequireQualifiedService(t *testing.T) {
	t.Parallel()
	var empty *client.Bundle
	if service, err := empty.AccountSettings(); service != nil || !errors.Is(err, client.ErrAccountSettings) {
		t.Fatal("nil bundle exposed adopted-account settings")
	}
	for _, withGetter := range []bool{false, true} {
		options := simpleTLSConfig()
		fixture := protectedSession(options)
		var session client.Session = fixture
		if withGetter {
			session = &accountSettingsSession{sessionFixture: fixture}
		}
		dials := 0
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
			dials++
			return session, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if service, err := bundle.AccountSettings(); service != nil || !errors.Is(err, client.ErrAccountSettings) {
			t.Fatal("unavailable account-settings service was not rejected")
		}
		if err := bundle.Close(); err != nil {
			t.Fatal(err)
		}
		if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
			t.Fatal("service lookup changed the qualified connection or its lifetime")
		}
	}
}
