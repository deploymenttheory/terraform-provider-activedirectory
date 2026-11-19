// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

// ValidateConfig rejects malformed known inputs while allowing Terraform to resolve unknown references before invocation.
func (a *AccountUnlockAction) ValidateConfig(ctx context.Context, req action.ValidateConfigRequest, resp *action.ValidateConfigResponse) {
	var model AccountUnlockActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateTarget(model, true)...)
}
