// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import "github.com/hashicorp/terraform-plugin-framework/types"

// ServiceAccountDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type ServiceAccountDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
}
