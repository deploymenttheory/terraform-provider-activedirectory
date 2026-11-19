// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/accounts/account_settings/mocks"
)

// unitProviderFactories registers only this read-only account settings surface over explicit synthetic SDK callbacks.
func unitProviderFactories(t *testing.T, service client.AccountSettings) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{DataSources: []func() datasource.DataSource{NewAccountSettingsDataSource}}, featuremocks.Factory(service))
}
