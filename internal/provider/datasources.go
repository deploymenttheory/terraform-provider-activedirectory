// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/group"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/groups"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_unit"
	organizationalUnits "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_units"
)

// DataSources explicitly registers the data sources included in the production provider.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		organizationalUnit.NewOrganizationalUnitDataSource,
		organizationalUnits.NewOrganizationalUnitsDataSource,
		single.NewGroupDataSource,
		plural.NewGroupsDataSource,
	}
}
