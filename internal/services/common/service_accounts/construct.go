// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_accounts

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// SearchModel defines explicit bounded discovery without an inferred directory partition or PowerShell filter parsing.
type SearchModel struct {
	BaseDN      types.String `tfsdk:"base_dn"`
	LDAPFilter  types.String `tfsdk:"ldap_filter"`
	Scope       types.String `tfsdk:"scope"`
	PageSize    types.Int64  `tfsdk:"page_size"`
	ResultLimit types.Int64  `tfsdk:"result_limit"`
}

// ConstructSearch validates the entire read request before SDK I/O.
func ConstructSearch(model SearchModel) (shared.SearchOptions, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	options := shared.SearchOptions{
		Base: model.BaseDN.ValueString(), Scope: shared.ScopeSubtree, PageSize: 256, ResultLimit: 1000,
	}
	if model.BaseDN.IsNull() || model.BaseDN.IsUnknown() {
		diagnostics.AddError("Unknown search base", "Select a known, explicit base_dn before searching service accounts.")
	} else if _, err := shared.IdentityByDN(options.Base); err != nil {
		diagnostics.AddError("Invalid search base", "base_dn must be an explicit distinguished name.")
	}
	if model.LDAPFilter.IsUnknown() || model.Scope.IsUnknown() || model.PageSize.IsUnknown() || model.ResultLimit.IsUnknown() {
		diagnostics.AddError("Unknown service account search input", "Scope, filter and limits must be known before directory discovery.")
	}
	if !model.LDAPFilter.IsNull() {
		options.Filter = model.LDAPFilter.ValueString()
	}
	if !model.Scope.IsNull() {
		switch model.Scope.ValueString() {
		case "base":
			options.Scope = shared.ScopeBase
		case "one_level":
			options.Scope = shared.ScopeOneLevel
		case "subtree":
		default:
			diagnostics.AddError("Invalid service account search scope", "scope must be base, one_level or subtree.")
		}
	}
	if !model.PageSize.IsNull() {
		if model.PageSize.ValueInt64() < 1 || model.PageSize.ValueInt64() > 1000 {
			diagnostics.AddError("Invalid service account page size", "page_size must be between 1 and 1000.")
		} else {
			options.PageSize = int(model.PageSize.ValueInt64())
		}
	}
	if !model.ResultLimit.IsNull() {
		if model.ResultLimit.ValueInt64() < 1 || model.ResultLimit.ValueInt64() > 10000 {
			diagnostics.AddError("Invalid service account result limit", "result_limit must be between 1 and 10000; zero does not enable an unbounded search.")
		} else {
			options.ResultLimit = int(model.ResultLimit.ValueInt64())
		}
	}
	if !diagnostics.HasError() {
		if err := options.Validate(); err != nil {
			diagnostics.AddError("Invalid service account search", "The SDK rejected the explicit search options.")
		}
	}
	return options, diagnostics
}

// ConstructLookup requires exactly one explicit GUID or DN and refuses unresolved inputs before reading.
func ConstructLookup(guid, dn types.String) (shared.Identity, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if guid.IsUnknown() || dn.IsUnknown() || guid.IsNull() == dn.IsNull() {
		diagnostics.AddError("Invalid service account lookup", "Supply exactly one known object_guid or dn.")
		return shared.Identity{}, diagnostics
	}
	var target shared.Identity
	var err error
	if !guid.IsNull() {
		var canonical string
		canonical, err = identity.CanonicalGUID(guid.ValueString())
		if err == nil {
			target, err = shared.IdentityByGUID(canonical)
		}
	} else {
		target, err = shared.IdentityByDN(dn.ValueString())
	}
	if err != nil {
		diagnostics.AddError("Invalid service account lookup identity", "The supplied GUID or distinguished name is malformed.")
	}
	return target, diagnostics
}
