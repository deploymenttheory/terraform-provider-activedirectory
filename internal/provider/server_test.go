// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider_test

import (
	"context"
	"errors"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
)

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
			p.err = errors.New("private peer cleanup response")
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
