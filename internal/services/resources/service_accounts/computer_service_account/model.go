// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import "github.com/hashicorp/terraform-plugin-framework/types"

// ComputerServiceAccountResourceModel identifies one direct link while keeping mutable directory names computed.
type ComputerServiceAccountResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ComputerGUID types.String `tfsdk:"computer_guid"`
	AccountGUID  types.String `tfsdk:"service_account_guid"`
	ComputerDN   types.String `tfsdk:"computer_dn"`
	AccountDN    types.String `tfsdk:"service_account_dn"`
	IsAssociated types.Bool   `tfsdk:"is_associated"`
}
