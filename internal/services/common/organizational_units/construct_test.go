// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExplicitLookupAndQueryBounds(t *testing.T) {
	valid := SearchModel{
		BaseDN: types.StringValue("DC=unit,DC=invalid"), LDAPFilter: types.StringNull(), Scope: types.StringNull(),
		PageSize: types.Int64Null(), ResultLimit: types.Int64Null(),
	}
	options, diagnostics := ConstructSearch(valid)
	if diagnostics.HasError() || options.PageSize != 256 || options.ResultLimit != 1000 {
		t.Fatal("bounded defaults were lost")
	}
	for _, change := range []func(*SearchModel){
		func(m *SearchModel) { m.BaseDN = types.StringUnknown() },
		func(m *SearchModel) { m.BaseDN = types.StringValue("ldap://unit.invalid") },
		func(m *SearchModel) { m.Scope = types.StringValue("inferred") },
		func(m *SearchModel) { m.PageSize = types.Int64Value(1001) },
		func(m *SearchModel) { m.ResultLimit = types.Int64Value(0) },
		func(m *SearchModel) { m.ResultLimit = types.Int64Value(10001) },
		func(m *SearchModel) { m.LDAPFilter = types.StringUnknown() },
	} {
		model := valid
		change(&model)
		if _, diagnostics := ConstructSearch(model); !diagnostics.HasError() {
			t.Fatal("unsafe/unknown search was accepted")
		}
	}
	for _, pair := range [][2]types.String{
		{types.StringNull(), types.StringNull()},
		{types.StringValue("a3e9767a-200e-43dd-bb1d-e5ab400e5d6f"), types.StringValue("OU=fixture,DC=unit,DC=invalid")},
		{types.StringUnknown(), types.StringNull()},
		{types.StringValue("00000000-0000-0000-0000-000000000000"), types.StringNull()},
		{types.StringNull(), types.StringValue("fixture")},
	} {
		if _, diagnostics := ConstructLookup(pair[0], pair[1]); !diagnostics.HasError() {
			t.Fatal("invalid lookup reached identity construction")
		}
	}
}
