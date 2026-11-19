// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

// TestTerraformUnitGroupManagedLifecycle proves stable GUID location changes, reader/SPN intent, import and clear semantics.
func TestTerraformUnitGroupManagedLifecycle(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := strings.ReplaceAll(string(data), "\r\n", "\n")
	updated := strings.NewReplacer(`name                    = "initial"`, `name                    = "renamed"`, "OU=first,", "OU=second,",
		"first description", "second description", "S-1-5-21-1-2-3-1001", "S-1-5-21-1-2-3-1002", "HTTP/initial.unit.invalid", "HTTP/updated.unit.invalid").Replace(initial)
	cleared := strings.Replace(updated, "  description             = \"second description\"", "  clear_properties        = [\"description\"]", 1)
	cleared = strings.Replace(cleared, `["HTTP/updated.unit.invalid"]`, `[]`, 1)
	cleared = strings.Replace(cleared, `["S-1-5-21-1-2-3-1002"]`, `[]`, 1)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "kind", "group_managed"),
			resource.TestCheckResourceAttr(
				ResourceName+".test",
				"password_reader_sids.#",
				"1",
			),
			resource.TestCheckResourceAttr(ResourceName+".test", "managed_password_interval_days", "30"),
		)},
		{Config: initial, PlanOnly: true},
		{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "name", "renamed"), resource.TestCheckResourceAttr(ResourceName+".test", "description", "second description"))},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)}},
		},
		{Config: cleared, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckNoResourceAttr(ResourceName+".test", "description"),
			resource.TestCheckResourceAttr(
				ResourceName+".test",
				"service_principal_names.#",
				"0",
			),
			resource.TestCheckResourceAttr(ResourceName+".test", "password_reader_sids.#", "0"),
		)},
		{Config: cleared, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for operation, want := range map[string]int{"create": 1, "rename": 1, "move": 1, "update": 2, "password": 0, "delete": 1} {
		if directory.calls[operation] != want {
			t.Errorf("%s calls=%d; want%d", operation, directory.calls[operation], want)
		}
	}
}

// TestTerraformUnitUnownedConcurrentChange verifies rename readback cannot turn an unconfigured DNS observation into write intent.
func TestTerraformUnitUnownedConcurrentChange(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := strings.ReplaceAll(string(data), "\r\n", "\n")
	unmanaged := strings.Replace(initial, "  dns_host_name           = \"initial.unit.invalid\"\n", "", 1)
	renamed := strings.Replace(unmanaged, `name                    = "initial"`, `name                    = "renamed"`, 1)
	directory := newUnitDirectory()
	directory.concurrentDNS = true
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: unmanaged, PlanOnly: true},
		{Config: renamed, Check: resource.TestCheckResourceAttr(ResourceName+".test", "dns_host_name", "concurrent.unit.invalid")},
		{Config: renamed, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["rename"] != 1 || directory.calls["update"] != 0 {
		t.Fatal("an observed unowned value was written after rename")
	}
}

// TestTerraformUnitInvalidLaterInputPreventsEarlierRename proves complete preflight precedes mutation phases.
func TestTerraformUnitInvalidLaterInputPreventsEarlierRename(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := strings.ReplaceAll(string(data), "\r\n", "\n")
	invalid := strings.NewReplacer(`name                    = "initial"`, `name                    = "invalid"`, "S-1-5-21-1-2-3-1001", "not-a-principal-SID").Replace(initial)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial}, {Config: invalid, ExpectError: regexp.MustCompile("Invalid password reader SID")}, {Config: initial, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["rename"] != 0 || directory.calls["move"] != 0 || directory.calls["update"] != 0 {
		t.Fatal("invalid later policy input allowed an earlier mutation")
	}
}

// TestTerraformUnitKindReplacementDropsGroupCreationMetadata proves sMSA replacement does not inherit a prior gMSA interval or reader policy.
func TestTerraformUnitKindReplacementDropsGroupCreationMetadata(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := strings.ReplaceAll(string(data), "\r\n", "\n")
	standalone := strings.Replace(initial, `kind                    = "group_managed"`, `kind                    = "standalone"`, 1)
	standalone = strings.Replace(standalone, "  password_reader_sids    = [\"S-1-5-21-1-2-3-1001\"]\n", "", 1)
	standalone = strings.Replace(standalone, "  dns_host_name           = \"initial.unit.invalid\"\n", "", 1)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: standalone, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(ResourceName+".test", plancheck.ResourceActionDestroyBeforeCreate),
		}}, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "kind", "standalone"),
			resource.TestCheckNoResourceAttr(
				ResourceName+".test",
				"managed_password_interval_days",
			),
			resource.TestCheckNoResourceAttr(ResourceName+".test", "password_reader_sids"),
		)},
		{Config: standalone, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["create"] != 2 || directory.calls["password"] != 1 || directory.guid == fixtureGUID {
		t.Fatal("kind replacement did not create a fresh standalone identity with one SDK password stage")
	}
}
