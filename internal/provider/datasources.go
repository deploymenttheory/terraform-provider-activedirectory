// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	computer "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computer"
	computers "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computers"
	group "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/group"
	groups "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/groups/groups"
	groupMember "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_member"
	groupMemberRelationships "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_member_relationships"
	object "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/object"
	objects "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/objects"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_unit"
	organizationalUnits "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/organizational_units/organizational_units"
	serviceAccount "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_account"
	serviceAccounts "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_accounts"
	user "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/user"
	users "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/users"
)

// DataSources explicitly registers the included directory data sources on the production provider.
func (p *ActiveDirectoryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		groupMemberRelationships.NewGroupMemberRelationshipsDataSource,
		groupMember.NewGroupMemberDataSource,
		organizationalUnit.NewOrganizationalUnitDataSource,
		organizationalUnits.NewOrganizationalUnitsDataSource,
		user.NewUserDataSource,
		users.NewUsersDataSource,
		computer.NewComputerDataSource,
		computers.NewComputersDataSource,
		group.NewGroupDataSource,
		groups.NewGroupsDataSource,
		serviceAccount.NewServiceAccountDataSource,
		serviceAccounts.NewServiceAccountsDataSource,
		object.NewObjectDataSource,
		objects.NewObjectsDataSource,
	}
}
