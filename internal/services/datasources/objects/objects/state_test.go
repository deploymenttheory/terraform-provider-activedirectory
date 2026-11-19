// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

// TestIncompleteSearchRetainsUsablePrefix verifies the state that accompanies an incomplete or duplicate result diagnostic.
func TestIncompleteSearchRetainsUsablePrefix(t *testing.T) {
	t.Parallel()
	payload, err := os.ReadFile("tests/responses/object.json")
	if err != nil {
		t.Fatal(err)
	}
	object, err := mocks.DecodeObjectFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	model := common.SearchModel{
		BaseDN: types.StringValue("DC=unit,DC=invalid"), LDAPFilter: types.StringNull(), Scope: types.StringNull(),
		PageSize: types.Int64Null(), ResultLimit: types.Int64Null(), Properties: types.SetNull(types.StringType),
	}
	options, diagnostics := common.ConstructSearch(model)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	var definition datasource.SchemaResponse
	NewObjectsDataSource().Schema(t.Context(), datasource.SchemaRequest{}, &definition)
	objectType := definition.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	initial := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		initial[name] = tftypes.NewValue(attributeType, nil)
	}
	for _, duplicate := range []bool{false, true} {
		state := tfsdk.State{Schema: definition.Schema, Raw: tftypes.NewValue(objectType, initial)}
		result := shared.SearchResult{Objects: []shared.Object{object}}
		if duplicate {
			result.Objects = append(result.Objects, object)
			result.Complete = true
		}
		diagnostics := mapState(t.Context(), &state, model, options, result)
		if diagnostics.HasError() != duplicate {
			t.Fatal("duplicate identity did not produce its own completeness diagnostic")
		}
		var rows types.List
		var complete types.Bool
		if state.GetAttribute(t.Context(), path.Root("objects"), &rows).HasError() ||
			state.GetAttribute(t.Context(), path.Root("complete"), &complete).HasError() || rows.IsNull() || len(rows.Elements()) != 1 || complete.ValueBool() {
			t.Fatal("a failed or duplicate search discarded the usable prefix or claimed complete absence")
		}
	}
}
