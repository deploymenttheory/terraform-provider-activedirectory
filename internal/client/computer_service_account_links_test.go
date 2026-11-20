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

type computerAssociationSession struct {
	*sessionFixture
	service client.ComputerServiceAccountLinks
}

func (s *computerAssociationSession) ComputerServiceAccountService() client.ComputerServiceAccountLinks {
	return s.service
}

type computerAssociationContract = client.ComputerServiceAccountLinks

type computerAssociationOperations struct{ computerAssociationContract }

// TestComputerServiceAccountLinksShareQualifiedSession verifies lookup neither substitutes the service nor opens another connection.
func TestComputerServiceAccountLinksShareQualifiedSession(t *testing.T) {
	t.Parallel()
	options := simpleTLSConfig()
	fixture := protectedSession(options)
	operations := &computerAssociationOperations{}
	session := &computerAssociationSession{sessionFixture: fixture, service: operations}
	dials := 0
	bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
		dials++
		return session, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		service, err := bundle.ComputerServiceAccountLinks()
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

// TestComputerServiceAccountLinksRequireQualifiedService refuses absent capabilities without opening an additional session.
func TestComputerServiceAccountLinksRequireQualifiedService(t *testing.T) {
	t.Parallel()
	var empty *client.Bundle
	if service, err := empty.ComputerServiceAccountLinks(); service != nil || !errors.Is(err, client.ErrComputerServiceAccountLinks) {
		t.Fatal("nil bundle exposed computerAssociation operations")
	}
	for _, withGetter := range []bool{false, true} {
		options := simpleTLSConfig()
		fixture := protectedSession(options)
		var session client.Session = fixture
		if withGetter {
			session = &computerAssociationSession{sessionFixture: fixture}
		}
		dials := 0
		bundle, err := client.OpenWithDial(context.Background(), options, func(context.Context, config.Config) (client.Session, error) {
			dials++
			return session, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if service, err := bundle.ComputerServiceAccountLinks(); service != nil || !errors.Is(err, client.ErrComputerServiceAccountLinks) {
			t.Fatal("unavailable computerAssociation service was not rejected")
		}
		if err := bundle.Close(); err != nil {
			t.Fatal(err)
		}
		if dials != 1 || fixture.closed != 1 || fixture.reads != 1 {
			t.Fatal("service lookup changed the qualified connection or its lifetime")
		}
	}
}
