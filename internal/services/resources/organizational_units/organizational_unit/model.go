// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import "github.com/hashicorp/terraform-plugin-framework/types"

// OrganizationalUnitResourceModel separates stable GUID identity, mutable location and managed properties.
type OrganizationalUnitResourceModel struct {
	ID                              types.String `tfsdk:"id"`
	Name                            types.String `tfsdk:"name"`
	ParentDN                        types.String `tfsdk:"parent_dn"`
	DistinguishedName               types.String `tfsdk:"distinguished_name"`
	Description                     types.String `tfsdk:"description"`
	DisplayName                     types.String `tfsdk:"display_name"`
	City                            types.String `tfsdk:"city"`
	Country                         types.String `tfsdk:"country"`
	State                           types.String `tfsdk:"state"`
	StreetAddress                   types.String `tfsdk:"street_address"`
	PostalCode                      types.String `tfsdk:"postal_code"`
	ManagedBy                       types.String `tfsdk:"managed_by"`
	ProtectedFromAccidentalDeletion types.Bool   `tfsdk:"protected_from_accidental_deletion"`
	RecoveryDN                      types.String `tfsdk:"recovery_dn"`
	ClearProperties                 types.Set    `tfsdk:"clear_properties"`
}
