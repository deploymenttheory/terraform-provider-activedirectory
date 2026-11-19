// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/groups/group/mocks"
)

// unitProviderFactories registers only the feature under test over fresh synthetic authenticated sessions.
func unitProviderFactories(t *testing.T, service client.Groups) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{Resources: []func() resource.Resource{NewGroupResource}}, featuremocks.Factory(service))
}
