// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import "github.com/hashicorp/terraform-plugin-framework/types"

// OrganizationalUnitDataSourceModel contains the two explicit, mutually exclusive lookup inputs.
type OrganizationalUnitDataSourceModel struct {
	ObjectGUID types.String `tfsdk:"object_guid"`
	DN         types.String `tfsdk:"dn"`
}
