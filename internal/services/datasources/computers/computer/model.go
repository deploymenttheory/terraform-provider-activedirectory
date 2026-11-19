// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import "github.com/hashicorp/terraform-plugin-framework/types"

// ComputerDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type ComputerDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
}
