// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestConstructObjectSearchKeepsExplicitBounds verifies an object-class-independent search with selected binary properties.
func TestConstructObjectSearchKeepsExplicitBounds(t *testing.T) {
	t.Parallel()
	model := SearchModel{
		BaseDN: types.StringValue("OU=owned,DC=unit,DC=invalid"), LDAPFilter: types.StringValue("(objectClass=contact)"),
		Scope: types.StringValue("one_level"), PageSize: types.Int64Value(20), ResultLimit: types.Int64Value(21),
		Properties: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("jpegPhoto"), types.StringValue("JPEGPHOTO")}),
	}
	options, diagnostics := ConstructSearch(model)
	if diagnostics.HasError() || options.Base != model.BaseDN.ValueString() || options.Filter != "(objectClass=contact)" ||
		options.Scope != shared.ScopeOneLevel || options.PageSize != 20 || options.ResultLimit != 21 {
		t.Fatal("generic discovery changed the explicit scope, class filter or bounds")
	}
	if !slices.Equal(options.Properties, []string{"distinguishedname", "jpegphoto", "name", "objectclass", "objectguid"}) {
		t.Fatal("raw property selection lost identity fields or repeated a case-insensitive attribute")
	}
	model.Scope, model.PageSize, model.ResultLimit = types.StringNull(), types.Int64Null(), types.Int64Null()
	options, diagnostics = ConstructSearch(model)
	if diagnostics.HasError() || options.Scope != shared.ScopeSubtree || options.PageSize != 256 || options.ResultLimit != 1000 {
		t.Fatal("omitted limits did not retain bounded discovery defaults")
	}
}

// TestConstructObjectSearchRefusesUnknownOrUnboundedIntent prevents directory reads from unresolved Terraform inputs.
func TestConstructObjectSearchRefusesUnknownOrUnboundedIntent(t *testing.T) {
	t.Parallel()
	for name, modify := range map[string]func(*SearchModel){
		"unknown base":       func(m *SearchModel) { m.BaseDN = types.StringUnknown() },
		"missing base":       func(m *SearchModel) { m.BaseDN = types.StringNull() },
		"unknown filter":     func(m *SearchModel) { m.LDAPFilter = types.StringUnknown() },
		"unknown scope":      func(m *SearchModel) { m.Scope = types.StringUnknown() },
		"unknown page":       func(m *SearchModel) { m.PageSize = types.Int64Unknown() },
		"zero limit":         func(m *SearchModel) { m.ResultLimit = types.Int64Value(0) },
		"negative page":      func(m *SearchModel) { m.PageSize = types.Int64Value(-1) },
		"large page":         func(m *SearchModel) { m.PageSize = types.Int64Value(1001) },
		"large limit":        func(m *SearchModel) { m.ResultLimit = types.Int64Value(10001) },
		"unknown properties": func(m *SearchModel) { m.Properties = types.SetUnknown(types.StringType) },
	} {
		t.Run(name, func(t *testing.T) {
			model := SearchModel{
				BaseDN: types.StringValue("OU=owned,DC=unit,DC=invalid"), Scope: types.StringNull(),
				LDAPFilter: types.StringNull(), PageSize: types.Int64Null(), ResultLimit: types.Int64Null(), Properties: types.SetNull(types.StringType),
			}
			modify(&model)
			if _, diagnostics := ConstructSearch(model); !diagnostics.HasError() {
				t.Fatal("unsafe or unresolved discovery intent reached a usable request")
			}
		})
	}
}

// TestPropertySelectionRefusesInventedVisibility preserves explicit selection instead of treating a wildcard as complete evidence.
func TestPropertySelectionRefusesInventedVisibility(t *testing.T) {
	t.Parallel()
	for _, element := range []attr.Value{
		types.StringUnknown(), types.StringNull(), types.StringValue("*"),
		types.StringValue("1.1"), types.StringValue("member;range=0-1499"),
	} {
		properties := types.SetValueMust(types.StringType, []attr.Value{element})
		if selected, diagnostics := ConstructProperties(properties); selected != nil || !diagnostics.HasError() {
			t.Fatal("unresolved or ambiguous raw attribute selection was accepted")
		}
	}
}

// TestObjectLookupRequiresOneExplicitIdentity rejects account-name/SID inference for generic directory objects.
func TestObjectLookupRequiresOneExplicitIdentity(t *testing.T) {
	t.Parallel()
	guid := types.StringValue("dc93656c-f745-4926-9bbb-31e7bbecb68b")
	dn := types.StringValue("CN=contact,OU=owned,DC=unit,DC=invalid")
	for _, input := range [][2]types.String{{guid, types.StringNull()}, {types.StringNull(), dn}} {
		if _, diagnostics := ConstructLookup(input[0], input[1]); diagnostics.HasError() {
			t.Fatal("an explicit generic object identity was rejected")
		}
	}
	for _, input := range [][2]types.String{
		{guid, dn},
		{types.StringNull(), types.StringNull()},
		{types.StringUnknown(), types.StringNull()},
		{types.StringValue("S-1-5-21-1234"), types.StringNull()},
	} {
		if _, diagnostics := ConstructLookup(input[0], input[1]); !diagnostics.HasError() {
			t.Fatal("ambiguous or inferred object identity was accepted")
		}
	}
}
