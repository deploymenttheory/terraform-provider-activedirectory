// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computer"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/computers/computers"
	discovery "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/computers/computer"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/computers/computer/mocks"
)

// unitProviderFactories registers only the computer feature over explicit synthetic SDK callbacks.
func unitProviderFactories(t *testing.T, service client.Computers) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{
		Resources:     []func() resource.Resource{NewComputerResource},
		DataSources:   []func() datasource.DataSource{single.NewComputerDataSource, plural.NewComputersDataSource},
		ListResources: []func() list.ListResource{discovery.NewComputerListResource},
	}, featuremocks.Factory(service))
}
