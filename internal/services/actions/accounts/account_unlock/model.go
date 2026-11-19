// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import "github.com/hashicorp/terraform-plugin-framework/types"

// AccountUnlockActionModel selects one stable account identity on one explicitly acknowledged DC.
type AccountUnlockActionModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	Server     types.String `tfsdk:"server"`
}
