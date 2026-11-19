// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"

	group "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/groups/group"
	object "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/objects/object"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/organizational_units/organizational_unit"
	user "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/users/user"
)

var _ provider.ProviderWithListResources = &ActiveDirectoryProvider{}

// ListResources explicitly registers the included directory list-resources on the production provider.
func (p *ActiveDirectoryProvider) ListResources(_ context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		organizationalUnit.NewOrganizationalUnitListResource,
		user.NewUserListResource,
		group.NewGroupListResource,
		object.NewObjectListResource,
	}
}
