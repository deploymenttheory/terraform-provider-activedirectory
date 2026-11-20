// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	lookup "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/accounts/account_settings"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/accounts/account_settings/mocks"
)

// unitProviderFactories isolates exactly the adopted-account resource and its read-only data source.
func unitProviderFactories(t *testing.T, service client.AccountSettings) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{
		Resources:   []func() resource.Resource{NewAccountSettingsResource},
		DataSources: []func() datasource.DataSource{lookup.NewAccountSettingsDataSource},
	}, featuremocks.Factory(service))
}
