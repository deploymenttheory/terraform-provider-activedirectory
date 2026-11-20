// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	managed "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
)

// unitProviderFactories isolates this feature and its explicitly required managed-resource identity schema.
func unitProviderFactories(t *testing.T, units client.Memberships) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(
		t,
		core.Registry{ListResources: []func() list.ListResource{NewGroupMemberListResource}, Resources: []func() resource.Resource{managed.NewGroupMemberResource}},
		mocks.Factory(units),
	)
}
