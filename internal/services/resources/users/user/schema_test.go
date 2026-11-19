// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSchemaKeepsPasswordWriteOnlyAndModelAligned(t *testing.T) {
	definition := userSchema()
	password, ok := definition.Attributes["password_wo"].(schema.StringAttribute)
	if !ok || !password.WriteOnly || !password.Sensitive || !password.Optional || password.Computed {
		t.Fatal("password input can be persisted or inferred from state")
	}
	model := creationModel()
	model.ID = types.StringValue("dc93656c-f745-4926-9bbb-31e7bbecb68b")
	model.PasswordWO = types.StringNull()
	state := tfsdk.State{Schema: definition}
	if diagnostics := state.Set(t.Context(), model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
}
