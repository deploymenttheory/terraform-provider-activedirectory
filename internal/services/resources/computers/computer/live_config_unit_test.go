// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

// TestTerraformUnitComputerAcceptanceConfiguration replays the embedded acceptance HCL using only synthetic SDK callbacks and credentials.
func TestTerraformUnitComputerAcceptanceConfiguration(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("AD_HOST", "dc.unit.invalid")
	t.Setenv("AD_USERNAME", "fixture@unit.invalid")
	t.Setenv("AD_PASSWORD", "synthetic-bind-password")
	t.Setenv("TF_VAR_computer_password", firstSecret)
	lab := &computerLab{
		parentDN: "OU=first,DC=unit,DC=invalid",
		prefix:   "tfad-cp-25-20261119t163018942z-123456abcdef",
		marker:   "tfad-owned:synthetic:windows",
	}
	moveDN := "OU=second," + lab.parentDN
	initial := lab.computerConfig(t, "initial", lab.parentDN, "activedirectory", `description = "first description"
 kerberos_encryption_types = ["aes128", "aes256"]
 compound_identity_supported = false`, "password_wo = var.computer_password\n password_wo_version = 1", true, "0")
	updated := lab.computerConfig(t, "renamed", moveDN, "activedirectory", `description = "second description"
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 change_password_at_logon = true`, "password_wo = var.computer_password\n password_wo_version = 2", true, "9223372036854775807")
	cleared := lab.computerConfig(t, "renamed", moveDN, "activedirectory", `clear_properties = ["description"]
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 change_password_at_logon = true`, "password_wo = var.computer_password\n password_wo_version = 2", true, "9223372036854775807")
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"))

	directory := newUnitDirectory(t)
	control := accounts.AccountControlPatch{}
	for _, field := range boolProperties(&directory.model, &control) {
		*field.value = types.BoolValue(false)
	}
	directory.model.ChangePasswordAtLogon = types.BoolValue(true)
	sid, err := shared.ParseSID("S-1-5-21-1111111111-2222222222-3333333333-1234")
	if err != nil {
		t.Fatal(err)
	}
	directory.sid = &sid
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: initial, PlanOnly: true},
		{Config: updated, PreConfig: func() { t.Setenv("TF_VAR_computer_password", secondSecret) }},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)}},
		{Config: cleared, Check: resource.TestCheckNoResourceAttr(ResourceName+".test", "description")},
		{Config: cleared, PlanOnly: true},
	}})
}
