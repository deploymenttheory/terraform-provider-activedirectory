// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Registry defines the service constructors available to one provider instance.
// Feature tests register only their own services and explicit fixture dependencies.
type Registry struct {
	Resources     []func() resource.Resource
	DataSources   []func() datasource.DataSource
	ListResources []func() list.ListResource
	Actions       []func() action.Action
}

// NewWithRegistry creates an isolated provider core without constructing services or opening clients.
// Constructor slices are copied so callers cannot alter another provider's registrations.
func NewWithRegistry(version string, registry Registry) *ActiveDirectoryProvider {
	return &ActiveDirectoryProvider{
		version: version,
		registry: Registry{
			Resources:     slices.Clone(registry.Resources),
			DataSources:   slices.Clone(registry.DataSources),
			ListResources: slices.Clone(registry.ListResources),
			Actions:       slices.Clone(registry.Actions),
		},
	}
}

// Resources returns an independent slice of this provider's resource constructors.
func (p *ActiveDirectoryProvider) Resources(_ context.Context) []func() resource.Resource {
	return slices.Clone(p.registry.Resources)
}

// DataSources returns an independent slice of this provider's data source constructors.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return slices.Clone(p.registry.DataSources)
}

// ListResources returns an independent slice of this provider's discovery constructors.
func (p *ActiveDirectoryProvider) ListResources(_ context.Context) []func() list.ListResource {
	return slices.Clone(p.registry.ListResources)
}

// Actions returns an independent slice of this provider's action constructors.
func (p *ActiveDirectoryProvider) Actions(_ context.Context) []func() action.Action {
	return slices.Clone(p.registry.Actions)
}
