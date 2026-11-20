// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func unitConfiguration(t *testing.T, settings string) string {
	t.Helper()
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Replace(string(data), "# __SETTINGS__", settings, 1)
}

// TestTerraformUnitAccountSettingsLifecycle verifies real adoption, updates, import, drift, no-change and relinquish behavior.
func TestTerraformUnitAccountSettingsLifecycle(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := newUnitDirectory(t)
	initial := unitConfiguration(t, `enabled = true
  account_not_delegated = true
  expires_at = "2031-04-05T11:34:56Z"`)
	clear := unitConfiguration(t, `enabled = true
  never_expires = true`)
	omitted := unitConfiguration(t, "")
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks()), Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "true"),
			resource.TestCheckResourceAttr(ResourceName+".test", "account_not_delegated", "true"),
			resource.TestCheckResourceAttr(ResourceName+".test", "never_expires", "false"),
			resource.TestCheckResourceAttrPair("data."+ResourceName+".by_dn", "id", ResourceName+".test", "id"),
		)},
		{Config: initial, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)}},
		},
		{
			Config: initial,
			PreConfig: func() {
				directory.mu.Lock()
				directory.accounts[fixtureGUID].flags["enabled"] = false
				directory.accounts[fixtureGUID].flags["home_directory_required"] = false
				directory.mu.Unlock()
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "true"),
				resource.TestCheckResourceAttr(ResourceName+".test", "home_directory_required", "false"),
			),
		},
		{
			Config: clear,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(ResourceName+".test", "never_expires", "true"),
				resource.TestCheckNoResourceAttr(ResourceName+".test", "expires_at"),
			),
		},
		{Config: clear, PlanOnly: true},
		{Config: omitted, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if !directory.accounts[fixtureGUID].exists || !directory.accounts[fixtureGUID].flags["enabled"] || directory.accounts[fixtureGUID].flags["home_directory_required"] ||
		directory.accounts[fixtureGUID].expiration != math.MaxInt64 {
		t.Fatal("relinquishment deleted/reset the existing account or unconfigured drift was overwritten")
	}
	if directory.calls["flags"] != 2 || directory.calls["set_expiration"] != 1 || directory.calls["clear_expiration"] != 1 {
		t.Fatalf("unexpected mutation counts: %v", directory.calls)
	}
	for _, patch := range directory.patches {
		for name := range patch {
			if name != "enabled" && name != "account_not_delegated" {
				t.Fatal("an observed unconfigured field entered an SDK write")
			}
		}
	}
}

// TestTerraformUnitTargetReplacementRelinquishesBothAccounts replaces only management identity, never the underlying directory objects.
func TestTerraformUnitTargetReplacementRelinquishesBothAccounts(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := newUnitDirectory(t)
	initial := unitConfiguration(t, "enabled = true")
	replaced := strings.Replace(initial, fixtureGUID, secondGUID, 1)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks()), Steps: []resource.TestStep{
		{Config: initial},
		{
			Config: replaced,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ResourceName+".test", plancheck.ResourceActionDestroyBeforeCreate)},
			},
			Check: resource.TestCheckResourceAttr(ResourceName+".test", "id", secondGUID),
		},
		{Config: replaced, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for _, row := range directory.accounts {
		if !row.exists || !row.flags["enabled"] {
			t.Fatal("target replacement deleted/reset either existing account")
		}
	}
	if directory.calls["flags"] != 2 {
		t.Fatal("target replacement replayed or omitted an account settings write")
	}
}

// TestTerraformUnitDeferredInvalidExpiryPreventsFlags exercises an apply-time dependency rather than only a constructor's unknown-value branch.
func TestTerraformUnitDeferredInvalidExpiryPreventsFlags(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := newUnitDirectory(t)
	configuration := unitConfiguration(t, `enabled = true
  expires_at = terraform_data.later.output`) + `
resource "terraform_data" "later" {
  input = "not-an-expiration"
}
`
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks()), Steps: []resource.TestStep{
		{Config: configuration, ExpectError: regexp.MustCompile("Invalid account expiration instant")},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["flags"] != 0 || directory.calls["set_expiration"] != 0 || directory.accounts[fixtureGUID].flags["enabled"] {
		t.Fatal("an invalid later resolved phase allowed the earlier enable write")
	}
}

// TestTerraformUnitUncertainEffectReconcilesWithoutReplay requires an independent fresh read before accepting an uncertain write's effect.
func TestTerraformUnitUncertainEffectReconcilesWithoutReplay(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := newUnitDirectory(t)
	service := directory.callbacks()
	update := service.UpdateControl
	service.UpdateControl = func(ctx context.Context, request accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
		result, err := update(ctx, request)
		if err != nil {
			return result, err
		}
		result.Mutation.State = shared.ObjectWriteUncertain
		result.Mutation.Verified = false
		return result, shared.ErrIncompleteResult
	}
	configuration := unitConfiguration(t, "enabled = true")
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: configuration, Check: resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "true")},
		{Config: configuration, PlanOnly: true},
	}})
	if directory.calls["flags"] != 1 || !directory.accounts[fixtureGUID].exists {
		t.Fatal("the independently reconciled write was replayed or relinquishment removed the account")
	}
}

// TestTerraformUnitUnresolvedWriteBlocksTaintedReplacement keeps private intent through an actual failed creation and later Terraform apply.
func TestTerraformUnitUnresolvedWriteBlocksTaintedReplacement(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := newUnitDirectory(t)
	service := directory.callbacks()
	service.UpdateControl = func(context.Context, accounts.AccountControlRequest) (accounts.AccountControlMutationResult, error) {
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.calls["flags"]++
		guid, _ := shared.ParseGUID(fixtureGUID)
		return accounts.AccountControlMutationResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteUncertain, TargetGUID: &guid}}, shared.ErrIncompleteResult
	}
	configuration := unitConfiguration(t, "enabled = true")
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: configuration, ExpectError: regexp.MustCompile("Account settings write needs reconciliation")},
		{Config: configuration, ExpectError: regexp.MustCompile("Account settings (write needs reconciliation|recovery required before relinquishment)")},
		{Config: configuration, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.accounts[fixtureGUID].exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["flags"] != 1 {
		t.Fatal("an unresolved effect was submitted again through Terraform replacement")
	}
}
