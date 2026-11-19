// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
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

// Schema defines provider configuration. Authentication is added by the SDK client foundation.
func (p *ActiveDirectoryProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Active Directory Domain Services using the Deployment Theory Active Directory provider.",
	}
}

// Configure prepares shared service configuration. The empty foundation does not open directory connections.
func (p *ActiveDirectoryProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}
