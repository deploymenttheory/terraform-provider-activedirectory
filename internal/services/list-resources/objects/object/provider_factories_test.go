// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	managed "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

// unitProviderFactories isolates this feature and its explicitly required managed-resource identity schema.
func unitProviderFactories(t *testing.T, units client.Objects) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(
		t,
		core.Registry{ListResources: []func() list.ListResource{NewObjectListResource}, Resources: []func() resource.Resource{managed.NewObjectResource}},
		mocks.Factory(units),
	)
}
