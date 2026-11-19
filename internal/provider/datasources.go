// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	group "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/group"
	groups "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/groups"
	object "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/object"
	objects "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/objects"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_unit"
	organizationalUnits "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_units"
	user "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/user"
	users "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/users"
)

// DataSources explicitly registers the included directory data sources on the production provider.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		organizationalUnit.NewOrganizationalUnitDataSource,
		organizationalUnits.NewOrganizationalUnitsDataSource,
		user.NewUserDataSource,
		users.NewUsersDataSource,
		group.NewGroupDataSource,
		groups.NewGroupsDataSource,
		object.NewObjectDataSource,
		objects.NewObjectsDataSource,
	}
}
