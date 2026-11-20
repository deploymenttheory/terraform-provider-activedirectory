// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import "github.com/hashicorp/terraform-plugin-framework/types"

// ComputerServiceAccountDataSourceModel selects one ordered GUID pair without owning the link.
type ComputerServiceAccountDataSourceModel struct {
	ComputerGUID types.String `tfsdk:"computer_guid"`
	AccountGUID  types.String `tfsdk:"service_account_guid"`
}
