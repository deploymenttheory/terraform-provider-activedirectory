// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

var (
	_ provider.Provider                  = &ActiveDirectoryProvider{}
	_ provider.ProviderWithActions       = &ActiveDirectoryProvider{}
	_ provider.ProviderWithListResources = &ActiveDirectoryProvider{}
)

// ActiveDirectoryProvider owns provider metadata and configuration shared by all service types.
type ActiveDirectoryProvider struct {
	version  string
	registry Registry
	mu       sync.Mutex
	clients  *client.Bundle
	open     client.Factory
	closed   bool
}

// NewActiveDirectoryProvider creates the shared core for one provider server.
func NewActiveDirectoryProvider(version string) *ActiveDirectoryProvider {
	return NewWithRegistry(version, Registry{})
}

// Metadata identifies the provider and reports the build version to Terraform.
func (p *ActiveDirectoryProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "activedirectory"
	resp.Version = p.version
}
