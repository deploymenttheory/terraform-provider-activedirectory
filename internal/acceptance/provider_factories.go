// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package acceptance provides shared Terraform test factories and explicit lab guards.
package acceptance

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

// ProviderFactories creates isolated Framework servers without importing production service registrations.
// The caller supplies either an SDK-boundary mock factory or the real factory after live prechecks.
func ProviderFactories(t *testing.T, registry providercore.Registry, factory client.Factory) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	if factory == nil {
		t.Fatal("an explicit SDK client factory is required")
	}
	return map[string]func() (tfprotov6.ProviderServer, error){
		"activedirectory": func() (tfprotov6.ProviderServer, error) {
			p := providercore.NewWithRegistry("test", registry, providercore.WithClientFactory(factory))
			t.Cleanup(func() {
				if err := p.Close(); err != nil {
					t.Error("provider client cleanup did not complete")
				}
			})
			return providercore.NewProtocol6Server(p), nil
		},
	}
}
