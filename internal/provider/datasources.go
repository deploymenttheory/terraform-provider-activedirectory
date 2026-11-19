// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/user"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/users"
)

// DataSources explicitly registers the data sources included in the production provider.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{single.NewUserDataSource, plural.NewUsersDataSource}
}
