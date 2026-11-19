// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func TestUpdateSelectsSemanticBitsAndExplicitClearWithoutClaimingOmission(t *testing.T) {
	current := creationModel()
	current.ID, current.Description = types.StringValue(fixtureGUID), types.StringValue("old")
	current.Enabled, current.AccountNotDelegated = types.BoolValue(true), types.BoolValue(true)
	current.CompoundIdentitySupported = types.BoolValue(true)
	current.KerberosEncryptionTypes = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("aes256")})
	plan := current
	plan.Description, plan.Enabled, plan.AccountNotDelegated = types.StringNull(), types.BoolValue(false), types.BoolNull()
	plan.KerberosEncryptionTypes, plan.CompoundIdentitySupported = types.SetValueMust(types.StringType, nil), types.BoolNull()
	plan.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("description")})
	target, _ := shared.IdentityByGUID(fixtureGUID)
	request, changed := constructUpdate(&current, &plan, target)
	if !changed || request.Control.Enabled == nil || *request.Control.Enabled || request.Control.AccountNotDelegated != nil ||
		request.Properties.KerberosEncryptionTypes == nil || *request.Properties.KerberosEncryptionTypes != 0 || request.Properties.CompoundIdentitySupported != nil ||
		request.Properties.Description != nil || len(request.Changes) != 1 || request.Changes[0].Kind != shared.AttributeClear || request.Changes[0].Attribute != "description" {
		t.Fatal("patch lost explicit zero/false/clear semantics or acquired an omitted setting")
	}
	retainAcceptedProperties(&current, &plan, request)
	if !current.Description.IsNull() || current.Enabled.ValueBool() || !current.AccountNotDelegated.ValueBool() ||
		!current.CompoundIdentitySupported.ValueBool() || len(current.KerberosEncryptionTypes.Elements()) != 0 {
		t.Fatal("accepted selected intent overwrote unrelated account settings")
	}
	if _, changed := constructUpdate(&current, &plan, target); changed {
		t.Fatal("acknowledged clearing or selected encryption was submitted again")
	}
}

func TestTerraformUnitInvalidLaterSettingsPreventEarlierRenameAndPassword(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	initial := computerFixtureConfig(t)
	invalid := strings.NewReplacer(`"initial"`, `"renamed"`, "password_wo_version = 1", "password_wo_version = 2").Replace(initial)
	invalid = strings.Replace(invalid, "  account_expires", "  kerberos_encryption_types = [\"aes1024\"]\n  account_expires", 1)
	directory := newUnitDirectory(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: invalid, ExpectError: regexp.MustCompile("Invalid computer encryption type")},
	}})
	if directory.calls["rename"] != 0 || directory.calls["password"] != 1 || directory.calls["update"] != 0 {
		t.Fatal("invalid later intent permitted an earlier mutation phase")
	}
}

func TestVerifiedNoopRequiresSameGUIDRawReadback(t *testing.T) {
	directory := newUnitDirectory(t)
	model := directory.model
	result := directory.mutation(t)
	result.State = shared.ObjectWriteNotAttempted
	if diagnostics := recordMutation(&model, result, nil); diagnostics.HasError() {
		t.Fatal("verified SDK no-op was reported as a failed write", diagnostics)
	}
	wrong, _ := shared.ParseGUID("ac93656c-f745-4926-9bbb-31e7bbecb68b")
	result.After.Attributes["objectguid"] = [][]byte{wrong.Bytes()}
	if diagnostics := recordMutation(&model, result, nil); !diagnostics.HasError() || model.ID.ValueString() != fixtureGUID {
		t.Fatal("conflicting raw readback authorized a no-op or changed stable identity")
	}
}
