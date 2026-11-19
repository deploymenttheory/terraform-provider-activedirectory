// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

// unitProviderFactories registers only this resource over explicitly supplied SDK callbacks.
func unitProviderFactories(t *testing.T, units client.OrganizationalUnits) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, providercore.Registry{Resources: []func() resource.Resource{NewOrganizationalUnitResource}}, featuremocks.Factory(units))
}
