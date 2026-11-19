// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	computer "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computer"
	computers "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computers"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_unit"
	organizationalUnits "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_units"
)

// DataSources explicitly registers the data sources included in the production provider.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		organizationalUnit.NewOrganizationalUnitDataSource, organizationalUnits.NewOrganizationalUnitsDataSource,
		computer.NewComputerDataSource, computers.NewComputersDataSource,
	}
}
