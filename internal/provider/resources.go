// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	computer "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/computers/computer"
	group "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/groups/group"
	groupMember "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member"
	object "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit"
	serviceAccount "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/service_account"
	user "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/users/user"
)

// Resources explicitly registers the included directory resources on the production provider.
func (p *ActiveDirectoryProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		groupMember.NewGroupMemberResource,
		organizationalUnit.NewOrganizationalUnitResource,
		user.NewUserResource,
		computer.NewComputerResource,
		group.NewGroupResource,
		serviceAccount.NewServiceAccountResource,
		object.NewObjectResource,
	}
}
