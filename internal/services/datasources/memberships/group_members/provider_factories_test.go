// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_members/mocks"
)

// unitProviderFactories registers only this read-only feature and its explicit SDK mock.
func unitProviderFactories(t *testing.T, units client.GroupMembershipQueries) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{DataSources: []func() datasource.DataSource{NewGroupMembersDataSource}}, mocks.Factory(units))
}
