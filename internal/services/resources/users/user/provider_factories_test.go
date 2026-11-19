// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/user"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/users/users"
	discovery "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/users/user"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/users/user/mocks"
)

// unitProviderFactories registers only the user resource over explicit synthetic SDK callbacks.
func unitProviderFactories(t *testing.T, service client.Users) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{
		Resources:     []func() resource.Resource{NewUserResource},
		DataSources:   []func() datasource.DataSource{single.NewUserDataSource, plural.NewUsersDataSource},
		ListResources: []func() list.ListResource{discovery.NewUserListResource},
	}, featuremocks.Factory(service))
}
