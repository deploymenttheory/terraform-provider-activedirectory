// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/groups/group"
	organizationalUnit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit"
)

// Resources explicitly registers the managed resources included in the production provider.
func (p *ActiveDirectoryProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{organizationalUnit.NewOrganizationalUnitResource, group.NewGroupResource}
}
