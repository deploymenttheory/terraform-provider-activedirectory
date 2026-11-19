// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"

	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

// ActiveDirectoryProvider combines the shared provider configuration with service registrations.
type ActiveDirectoryProvider struct {
	*providercore.ActiveDirectoryProvider
}

// NewActiveDirectoryProvider returns a factory that creates an isolated provider instance for each server.
func NewActiveDirectoryProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ActiveDirectoryProvider{
			ActiveDirectoryProvider: providercore.NewActiveDirectoryProvider(version),
		}
	}
}
