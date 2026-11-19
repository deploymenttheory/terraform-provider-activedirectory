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

type membershipSession struct {
	*sessionFixture
	service client.Memberships
}

func (s *membershipSession) MembershipService() client.Memberships { return s.service }

type membershipContract = client.Memberships

type membershipOperations struct{ membershipContract }

// TestMembershipsShareQualifiedSession verifies lookup neither substitutes the service nor opens another connection.
func TestMembershipsShareQualifiedSession(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	fixture := protectedSession(options)
	operations := &membershipOperations{}
	session := &membershipSession{sessionFixture: fixture, service: operations}
	dials := 0
	bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
		dials++
		return session, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		service, err := bundle.Memberships()
		if err != nil || service != operations {
			t.Fatal("direct-link operations did not retain the qualified service")
		}
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
		t.Fatal("direct-link surfaces did not share provider qualification and closure")
	}
}

// TestMembershipsRequireQualifiedService refuses absent capabilities without opening an additional session.
func TestMembershipsRequireQualifiedService(t *testing.T) {
	t.Parallel()
	var empty *client.Bundle
	if service, err := empty.Memberships(); service != nil || !errors.Is(err, client.ErrMemberships) {
		t.Fatal("nil bundle exposed membership operations")
	}
	for _, withGetter := range []bool{false, true} {
		options := simpleTLSConfig()
		fixture := protectedSession(options)
		var session client.Session = fixture
		if withGetter {
			session = &membershipSession{sessionFixture: fixture}
		}
		dials := 0
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
			dials++
			return session, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if service, err := bundle.Memberships(); service != nil || !errors.Is(err, client.ErrMemberships) {
			t.Fatal("unavailable membership service was not rejected")
		}
		if err := bundle.Close(); err != nil {
			t.Fatal(err)
		}
		if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
			t.Fatal("service lookup changed the qualified connection or its lifetime")
		}
	}
}
