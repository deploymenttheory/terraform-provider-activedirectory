// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
)

// unitProviderFactories registers only the direct-link resource over independently qualified synthetic sessions.
func unitProviderFactories(t *testing.T, service client.Memberships) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{Resources: []func() resource.Resource{NewGroupMemberResource}}, mocks.Factory(service))
}
