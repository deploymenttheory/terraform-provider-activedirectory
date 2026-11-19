// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import "github.com/hashicorp/terraform-plugin-framework/types"

// ObjectDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type ObjectDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
	Properties types.Set    `tfsdk:"properties"`
}
