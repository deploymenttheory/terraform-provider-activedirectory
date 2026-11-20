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

type groupQuerySession struct {
	*sessionFixture
	service client.GroupMembershipQueries
}

func (s *groupQuerySession) GroupMembershipQueryService() client.GroupMembershipQueries {
	return s.service
}

type groupQueryContract = client.GroupMembershipQueries

type groupQueryOperations struct{ groupQueryContract }

// TestGroupMembershipQueriesShareQualifiedSession verifies lookup neither substitutes the service nor opens another connection.
func TestGroupMembershipQueriesShareQualifiedSession(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	fixture := protectedSession(options)
	operations := &groupQueryOperations{}
	session := &groupQuerySession{sessionFixture: fixture, service: operations}
	dials := 0
	bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
		dials++
		return session, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		service, err := bundle.GroupMembershipQueries()
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

// TestGroupMembershipQueriesRequireQualifiedService refuses absent capabilities without opening an additional session.
func TestGroupMembershipQueriesRequireQualifiedService(t *testing.T) {
	t.Parallel()
	var empty *client.Bundle
	if service, err := empty.GroupMembershipQueries(); service != nil || !errors.Is(err, client.ErrGroupMembershipQueries) {
		t.Fatal("nil bundle exposed groupQuery operations")
	}
	for _, withGetter := range []bool{false, true} {
		options := simpleTLSConfig()
		fixture := protectedSession(options)
		var session client.Session = fixture
		if withGetter {
			session = &groupQuerySession{sessionFixture: fixture}
		}
		dials := 0
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
			dials++
			return session, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if service, err := bundle.GroupMembershipQueries(); service != nil || !errors.Is(err, client.ErrGroupMembershipQueries) {
			t.Fatal("unavailable groupQuery service was not rejected")
		}
		if err := bundle.Close(); err != nil {
			t.Fatal(err)
		}
		if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
			t.Fatal("service lookup changed the qualified connection or its lifetime")
		}
	}
}
