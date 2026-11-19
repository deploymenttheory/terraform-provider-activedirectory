// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_account"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_accounts"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/service_account/mocks"
)

// unitProviderFactories isolates service-account schemas over explicit SDK callbacks without live credentials.
func unitProviderFactories(t *testing.T, service client.ServiceAccounts) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{
		Resources:   []func() resource.Resource{NewServiceAccountResource},
		DataSources: []func() datasource.DataSource{single.NewServiceAccountDataSource, plural.NewServiceAccountsDataSource},
	}, featuremocks.Factory(service))
}
