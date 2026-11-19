// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider_test

import (
	"context"
	"errors"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	provider "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

var errPrivateCleanup = errors.New("private peer cleanup response")

type managedFixture struct {
	frameworkprovider.Provider
	closed bool
	err    error
}

func (p *managedFixture) Close() error { p.closed = true; return p.err }

func TestProtocolStopClosesClientsAndRedactsCleanupErrors(t *testing.T) {
	t.Parallel()
	for _, failure := range []bool{false, true} {
		p := &managedFixture{}
		if failure {
			p.err = errPrivateCleanup
		}
		server := provider.NewProtocol6Server(p)
		response, err := server.StopProvider(context.Background(), &tfprotov6.StopProviderRequest{})
		if err != nil || !p.closed {
			t.Fatalf("stop did not close provider clients: %v", err)
		}
		if failure && response.Error != "The directory session could not be closed safely." {
			t.Fatal("cleanup error was not sanitized")
		}
		if !failure && response.Error != "" {
			t.Fatal("successful cleanup returned an error")
		}
	}
}

func TestShutdownAdapterPreservesListAndActionRPCs(t *testing.T) {
	t.Parallel()
	p := provider.NewWithRegistry("test", provider.Registry{})
	server := provider.NewProtocol6Server(p)
	if _, ok := server.(tfprotov6.ProviderServerWithListResource); !ok {
		t.Error("shutdown adapter erased list-resource RPCs")
	}
	if _, ok := server.(tfprotov6.ProviderServerWithActions); !ok {
		t.Error("shutdown adapter erased action RPCs")
	}
}
