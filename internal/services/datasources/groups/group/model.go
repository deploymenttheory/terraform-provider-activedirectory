// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import "github.com/hashicorp/terraform-plugin-framework/types"

// GroupDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type GroupDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
}
