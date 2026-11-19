// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import "github.com/hashicorp/terraform-plugin-framework/types"

// UserResourceModel separates stable identity, mutable properties and write-only password input from its acknowledged version.
type UserResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	ParentDN               types.String `tfsdk:"parent_dn"`
	ObjectClass            types.String `tfsdk:"object_class"`
	DistinguishedName      types.String `tfsdk:"distinguished_name"`
	SID                    types.String `tfsdk:"sid"`
	RecoveryDN             types.String `tfsdk:"recovery_dn"`
	SAMAccountName         types.String `tfsdk:"sam_account_name"`
	UserPrincipalName      types.String `tfsdk:"user_principal_name"`
	GivenName              types.String `tfsdk:"given_name"`
	Surname                types.String `tfsdk:"surname"`
	DisplayName            types.String `tfsdk:"display_name"`
	Description            types.String `tfsdk:"description"`
	EmailAddress           types.String `tfsdk:"email_address"`
	OfficePhone            types.String `tfsdk:"office_phone"`
	MobilePhone            types.String `tfsdk:"mobile_phone"`
	Department             types.String `tfsdk:"department"`
	Company                types.String `tfsdk:"company"`
	Title                  types.String `tfsdk:"title"`
	Office                 types.String `tfsdk:"office"`
	City                   types.String `tfsdk:"city"`
	State                  types.String `tfsdk:"state"`
	StreetAddress          types.String `tfsdk:"street_address"`
	PostalCode             types.String `tfsdk:"postal_code"`
	HomeDirectory          types.String `tfsdk:"home_directory"`
	HomeDrive              types.String `tfsdk:"home_drive"`
	ProfilePath            types.String `tfsdk:"profile_path"`
	ScriptPath             types.String `tfsdk:"script_path"`
	ManagerDN              types.String `tfsdk:"manager_dn"`
	AccountExpires         types.Int64  `tfsdk:"account_expires"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	PasswordNotRequired    types.Bool   `tfsdk:"password_not_required"`
	PasswordNeverExpires   types.Bool   `tfsdk:"password_never_expires"`
	SmartcardLogonRequired types.Bool   `tfsdk:"smartcard_logon_required"`
	PasswordWO             types.String `tfsdk:"password_wo"`
	PasswordWOVersion      types.Int64  `tfsdk:"password_wo_version"`
	ClearProperties        types.Set    `tfsdk:"clear_properties"`
}

// String prevents accidental diagnostic formatting from disclosing write-only input.
func (UserResourceModel) String() string { return "UserResourceModel{redacted}" }

// GoString applies the same redaction to Go's detailed formatting form.
func (UserResourceModel) GoString() string { return "UserResourceModel{redacted}" }
