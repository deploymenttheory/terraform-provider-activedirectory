// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// SearchModel selects a bounded generic search and explicit raw properties without guessing a naming context or schema.
type SearchModel struct {
	BaseDN      types.String `tfsdk:"base_dn"`
	LDAPFilter  types.String `tfsdk:"ldap_filter"`
	Scope       types.String `tfsdk:"scope"`
	PageSize    types.Int64  `tfsdk:"page_size"`
	ResultLimit types.Int64  `tfsdk:"result_limit"`
	Properties  types.Set    `tfsdk:"properties"`
}

// ConstructSearch refuses unresolved inputs and unbounded discovery before any SDK call.
func ConstructSearch(model SearchModel) (shared.SearchOptions, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	options := shared.SearchOptions{Base: model.BaseDN.ValueString(), Scope: shared.ScopeSubtree, PageSize: 256, ResultLimit: 1000}
	if model.BaseDN.IsNull() || model.BaseDN.IsUnknown() {
		diagnostics.AddError("Unknown object search base", "Select a known, explicit base_dn before searching directory objects.")
	} else if _, err := shared.IdentityByDN(options.Base); err != nil {
		diagnostics.AddError("Invalid object search base", "base_dn must be an explicit distinguished name.")
	}
	if model.LDAPFilter.IsUnknown() || model.Scope.IsUnknown() || model.PageSize.IsUnknown() || model.ResultLimit.IsUnknown() {
		diagnostics.AddError("Unknown object search input", "Scope, filter and limits must be known before directory discovery.")
	}
	options.Filter = model.LDAPFilter.ValueString()
	if !model.Scope.IsNull() {
		switch model.Scope.ValueString() {
		case "base":
			options.Scope = shared.ScopeBase
		case "one_level":
			options.Scope = shared.ScopeOneLevel
		case "subtree":
		default:
			diagnostics.AddError("Invalid object search scope", "scope must be base, one_level or subtree.")
		}
	}
	if !model.PageSize.IsNull() {
		if size := model.PageSize.ValueInt64(); size >= 1 && size <= 1000 {
			options.PageSize = int(size)
		} else {
			diagnostics.AddError("Invalid object page size", "page_size must be between 1 and 1000.")
		}
	}
	if !model.ResultLimit.IsNull() {
		if limit := model.ResultLimit.ValueInt64(); limit >= 1 && limit <= 10000 {
			options.ResultLimit = int(limit)
		} else {
			diagnostics.AddError("Invalid object result limit", "result_limit must be between 1 and 10000; zero does not enable unbounded discovery.")
		}
	}
	properties, propertyDiagnostics := ConstructProperties(model.Properties)
	diagnostics.Append(propertyDiagnostics...)
	options.Properties = properties
	if !diagnostics.HasError() {
		if err := options.Validate(); err != nil {
			diagnostics.AddError("Invalid object search", "The SDK rejected the explicit search options.")
		}
	}
	return options, diagnostics
}

// ConstructProperties selects stable identity fields and explicitly requested raw attributes; wildcards never imply visibility.
func ConstructProperties(properties types.Set) ([]string, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	selected := map[string]bool{"objectguid": true, "distinguishedname": true, "name": true, "objectclass": true}
	if properties.IsUnknown() {
		diagnostics.AddError("Unknown raw properties", "Selected raw property names must be known before reading directory objects.")
		return nil, diagnostics
	}
	for _, element := range properties.Elements() {
		name, ok := element.(types.String)
		if !ok || name.IsUnknown() || name.IsNull() || !attributeName.MatchString(name.ValueString()) {
			diagnostics.AddError("Invalid raw property", "properties must contain known LDAP display names without wildcards, options or numeric aliases.")
			continue
		}
		selected[strings.ToLower(name.ValueString())] = true
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	result := make([]string, 0, len(selected))
	for name := range selected {
		result = append(result, name)
	}
	slices.Sort(result)
	return result, diagnostics
}

// ConstructLookup accepts exactly one explicit GUID or DN and preserves generic object identity rules.
func ConstructLookup(guid, dn types.String) (shared.Identity, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if guid.IsUnknown() || dn.IsUnknown() || guid.IsNull() == dn.IsNull() {
		diagnostics.AddError("Invalid object lookup", "Supply exactly one known object_guid or dn.")
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
		diagnostics.AddError("Invalid object lookup identity", "The supplied GUID or distinguished name is malformed.")
	}
	return target, diagnostics
}
