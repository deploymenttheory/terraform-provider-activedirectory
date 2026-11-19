// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestGroupSchemaMatchesModelAndExcludesCompetingOwnership(t *testing.T) {
	definition := groupSchema()
	model := creationModel()
	state := tfsdk.State{Schema: definition}
	if diagnostics := state.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal("group model and Framework schema do not agree", diagnostics)
	}
	for _, forbidden := range []string{"members", "member", "member_of", "group_type"} {
		if _, exists := definition.Attributes[forbidden]; exists {
			t.Fatal("group lifecycle exposed competing membership ownership or a raw flag mask")
		}
	}
}
