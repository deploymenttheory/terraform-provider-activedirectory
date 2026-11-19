// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func computerFixtureConfig(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestTerraformUnitPasswordResetOwnsOnlyExplicitLogonSetting(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name, expected, updates := "omitted", "false", 0
		if explicit {
			name, expected, updates = "explicit", "true", 1
		}
		t.Run(name, func(t *testing.T) {
			acceptance.IsolateUnitEnvironment(t)
			t.Setenv("TF_VAR_computer_password", firstSecret)
			withPassword := computerFixtureConfig(t)
			initial := strings.NewReplacer("  enabled             = true\n", "", "  password_wo         = var.computer_password\n", "",
				"  password_wo_version = 1\n", "").Replace(withPassword)
			rotated := strings.Replace(withPassword, "  enabled             = true\n", "", 1)
			if explicit {
				initial = strings.Replace(initial, "  account_expires", "  change_password_at_logon = true\n  account_expires", 1)
				rotated = strings.Replace(rotated, "  account_expires", "  change_password_at_logon = true\n  account_expires", 1)
			}
			directory := newUnitDirectory(t)
			t.Chdir(t.TempDir())
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
				{Config: initial, Check: resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "true")},
				{Config: initial, PlanOnly: true},
				{
					Config: rotated, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}},
					Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", expected),
						resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "1")),
				},
				{Config: rotated, PlanOnly: true},
			}})
			if directory.calls["password"] != 1 || directory.calls["update"] != updates {
				t.Fatal("password reset replayed or acquired an omitted logon-password setting")
			}
		})
	}
}

func TestTerraformUnitCreationRetainsRootCheckpointAfterLogonSettingFailure(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	config := strings.Replace(computerFixtureConfig(t), "  account_expires", "  change_password_at_logon = true\n  account_expires", 1)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	service.Update = func(_ context.Context, request computers.UpdateComputerRequest) (computers.ComputerMutationResult, error) {
		directory.mu.Lock()
		defer directory.mu.Unlock()
		if request.Control.ChangePasswordAtLogon == nil || !*request.Control.ChangePasswordAtLogon || request.Identity.String() != fixtureGUID {
			t.Fatal("post-create phase lost explicit logon-password intent or stable identity")
		}
		directory.calls["update"]++
		guid, _ := shared.ParseGUID(fixtureGUID)
		return computers.ComputerMutationResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteRejected, TargetGUID: &guid}}, shared.ErrObjectWritePermission
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: config, ExpectError: regexp.MustCompile("Computer mutation still unresolved")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{Config: config, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["create"] != 1 || directory.calls["password"] != 1 || directory.calls["update"] != 1 || directory.calls["delete"] != 0 {
		t.Fatal("a failed post-create phase allowed replacement, rollback or password replay")
	}
}
