// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

// unitProviderFactories registers only this actual managed resource and explicitly supplied synthetic SDK operations.
func unitProviderFactories(t *testing.T, service client.Objects) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, core.Registry{Resources: []func() resource.Resource{NewObjectResource}}, mocks.Factory(service))
}
