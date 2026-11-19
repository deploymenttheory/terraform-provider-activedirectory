// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import "github.com/hashicorp/terraform-plugin-framework/types"

// UserDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type UserDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
}
