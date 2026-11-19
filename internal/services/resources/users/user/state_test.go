// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var errSecretInArtifact = errors.New("write-only user secret entered a Terraform plan or state artifact")

func TestPersistStateClearsWriteOnlyValueBeforeSerialization(t *testing.T) {
	model := creationModel()
	model.PasswordWO = types.StringValue(firstSecret)
	model.PasswordWOVersion = types.Int64Value(3)
	state := tfsdk.State{Schema: userSchema()}
	if diagnostics := persistState(t.Context(), &state, nil, &model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if !model.PasswordWO.IsNull() || strings.Contains(state.Raw.String(), firstSecret) || model.PasswordWOVersion.ValueInt64() != 3 {
		t.Fatal(errSecretInArtifact)
	}
}
