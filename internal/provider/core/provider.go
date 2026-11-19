// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ provider.Provider                  = &ActiveDirectoryProvider{}
	_ provider.ProviderWithActions       = &ActiveDirectoryProvider{}
	_ provider.ProviderWithListResources = &ActiveDirectoryProvider{}
)

// ActiveDirectoryProvider owns provider metadata and configuration shared by all service types.
type ActiveDirectoryProvider struct {
	version string
}

// NewActiveDirectoryProvider creates the shared core for one provider server.
func NewActiveDirectoryProvider(version string) *ActiveDirectoryProvider {
	return &ActiveDirectoryProvider{version: version}
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

// Resources returns the managed resource constructors available in this provider build.
func (p *ActiveDirectoryProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

// DataSources returns the data source constructors available in this provider build.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// ListResources returns the discovery constructors available to Terraform query configurations.
func (p *ActiveDirectoryProvider) ListResources(_ context.Context) []func() list.ListResource {
	return nil
}

// Actions returns the explicit operation constructors available in this provider build.
func (p *ActiveDirectoryProvider) Actions(_ context.Context) []func() action.Action {
	return nil
}
