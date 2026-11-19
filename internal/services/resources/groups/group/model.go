// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import "github.com/hashicorp/terraform-plugin-framework/types"

// GroupResourceModel owns individual typed group properties; membership edges belong to separate resources.
type GroupResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ParentDN          types.String `tfsdk:"parent_dn"`
	SAMAccountName    types.String `tfsdk:"sam_account_name"`
	Category          types.String `tfsdk:"category"`
	Scope             types.String `tfsdk:"scope"`
	Description       types.String `tfsdk:"description"`
	DisplayName       types.String `tfsdk:"display_name"`
	HomePage          types.String `tfsdk:"home_page"`
	ManagedByDN       types.String `tfsdk:"managed_by_dn"`
	DistinguishedName types.String `tfsdk:"distinguished_name"`
	SID               types.String `tfsdk:"sid"`
	RecoveryDN        types.String `tfsdk:"recovery_dn"`
	ClearProperties   types.Set    `tfsdk:"clear_properties"`
}
