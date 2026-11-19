// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

var _ provider.ProviderWithActions = &ActiveDirectoryProvider{}

// Actions explicitly registers the operations available through Terraform action blocks.
func (p *ActiveDirectoryProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{}
}
