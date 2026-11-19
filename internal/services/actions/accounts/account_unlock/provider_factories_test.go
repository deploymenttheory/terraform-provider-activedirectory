// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/actions/accounts/account_unlock/mocks"
)

// unitProviderFactories registers only this action and a synthetic SDK account service.
func unitProviderFactories(t *testing.T, service client.Accounts) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, providercore.Registry{Actions: []func() action.Action{NewAccountUnlockAction}}, featuremocks.Factory(service))
}
