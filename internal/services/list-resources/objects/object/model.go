// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import "github.com/hashicorp/terraform-plugin-framework/types"

// ObjectListModel follows the Framework list configuration's ordered collection types.
type ObjectListModel struct {
	BaseDN      types.String `tfsdk:"base_dn"`
	LDAPFilter  types.String `tfsdk:"ldap_filter"`
	Scope       types.String `tfsdk:"scope"`
	PageSize    types.Int64  `tfsdk:"page_size"`
	ResultLimit types.Int64  `tfsdk:"result_limit"`
	Properties  types.List   `tfsdk:"properties"`
}
