// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestComputerSchemaAlignsModelAndKeepsSecretsWriteOnly(t *testing.T) {
	definition := computerSchema()
	password, ok := definition.Attributes["password_wo"].(schema.StringAttribute)
	if !ok || !password.WriteOnly || !password.Sensitive || !password.Optional || password.Computed {
		t.Fatal("computer account secret can enter ordinary plan/state")
	}
	model := creationModel()
	model.PasswordWO = types.StringNull()
	state := tfsdk.State{Schema: definition}
	if diagnostics := state.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal("computer Framework model/schema mismatch", diagnostics)
	}
	for _, forbidden := range []string{"domain_join", "domain_admin_password", "user_account_control", "raw_encryption_mask", "recursive"} {
		if _, exists := definition.Attributes[forbidden]; exists {
			t.Fatal("computer schema exposed host management, an account-type mask or implicit recursive ownership")
		}
	}
	if !strings.Contains(definition.Description, "never joins or configures a machine") {
		t.Fatal("computer resource no longer distinguishes object creation from machine domain join")
	}
}
