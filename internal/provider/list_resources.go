// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

var _ provider.ProviderWithListResources = &ActiveDirectoryProvider{}

// ListResources explicitly registers discovery services for Terraform query and import workflows.
func (p *ActiveDirectoryProvider) ListResources(_ context.Context) []func() list.ListResource {
	return []func() list.ListResource{}
}
