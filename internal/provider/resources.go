// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	accountSettings "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/accounts/account_settings"
	computer "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/computers/computer"
	group "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/groups/group"
	object "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit"
	user "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/users/user"
)

// Resources explicitly registers the included directory resources on the production provider.
func (p *ActiveDirectoryProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		organizationalUnit.NewOrganizationalUnitResource,
		user.NewUserResource,
		computer.NewComputerResource,
		group.NewGroupResource,
		object.NewObjectResource,
		accountSettings.NewAccountSettingsResource,
	}
}
