// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func TestTerraformUnitCompleteComputerObservationAndPolicyChanges(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	initial := strings.Replace(
		computerFixtureConfig(t),
		"  account_expires",
		"  kerberos_encryption_types = [\"aes128\", \"aes256\"]\n  compound_identity_supported = false\n  account_expires",
		1,
	)
	updated := strings.NewReplacer(`"initial"`, `"renamed"`, "OU=first,DC=unit,DC=invalid", "OU=second,DC=unit,DC=invalid",
		`["aes128", "aes256"]`, `["aes256"]`, "compound_identity_supported = false", "compound_identity_supported = true\n  change_password_at_logon = true",
		"password_wo_version = 1", "password_wo_version = 2", "account_expires     = 0", "account_expires     = 9223372036854775807").Replace(initial)
	directory := newUnitDirectory(t)
	control := accounts.AccountControlPatch{}
	for _, field := range boolProperties(&directory.model, &control) {
		*field.value = types.BoolValue(false)
	}
	directory.model.ChangePasswordAtLogon = types.BoolValue(true)
	query, err := os.ReadFile("tests/terraform/unit/list-resource.tfquery.hcl")
	if err != nil {
		t.Fatal(err)
	}
	cleared := strings.Replace(updated, "  description         = \"first description\"\n", "  clear_properties = [\"description\"]\n", 1)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: initial, PlanOnly: true},
		{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "kerberos_encryption_types.#", "1"),
			resource.TestCheckResourceAttr(ResourceName+".test", "compound_identity_supported", "true"),
			resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "true"),
			resource.TestCheckResourceAttr(ResourceName+".test", "account_expires", "9223372036854775807"))},
		{Config: updated, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true},
		{Query: true, Config: string(query), QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".test", 1)}},
		{Config: cleared, Check: resource.TestCheckNoResourceAttr(ResourceName+".test", "description")},
		{Config: cleared, PlanOnly: true},
	}})
}
