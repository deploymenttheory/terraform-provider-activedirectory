// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

// TestTerraformUnitObjectLifecycle exercises binary ownership, strict value operations, stable import, drift and accepted clears.
func TestTerraformUnitObjectLifecycle(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	updated := strings.NewReplacer(`"fixture"`, `"renamed"`, "OU=first,", "OU=second,", "Zmlyc3Q=", "c2Vjb25k", "/wCA", "AP+A",
		`["b2xk", "a2VlcA=="]`, `["bmV3"]`).Replace(initial)
	updated = strings.Replace(updated, "\n  properties", "\n  remove_values = { otherTelephone = [\"b2xk\"] }\n  properties", 1)
	cleared := regexp.MustCompile(`description\s*=\s*\["c2Vjb25k"\],?\s*`).ReplaceAllString(updated, "")
	cleared = strings.Replace(cleared, "\n  properties", "\n  clear_attributes = [\"description\"]\n  properties", 1)
	observedOnly := strings.Replace(cleared, `["telephoneNumber"]`, `["telephoneNumber", "displayName"]`, 1)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "attributes.thumbnailPhoto.*", "/wCA"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.thumbnailphoto.*", "/wCA"))},
		{Config: initial, PlanOnly: true},
		{Config: strings.Replace(initial, `"contact"`, `"CONTACT"`, 1)},
		{Config: initial},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true,
			ImportStateId:           `{"id":"` + fixtureGUID + `","attributes":["description","thumbnailPhoto"],"properties":["telephoneNumber"]}`,
			ImportStateVerifyIgnore: []string{"add_values", "observed_attributes", "attribute_status", "attribute_ranges"},
		},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)}},
		},
		{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "name", "renamed"),
			resource.TestCheckResourceAttr(ResourceName+".test", "parent_dn", "OU=second,DC=unit,DC=invalid"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", "a2VlcA=="),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", "bmV3"))},
		{Config: updated, PlanOnly: true},
		{Config: cleared, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckNoResourceAttr(ResourceName+".test", "observed_attributes.description"),
			resource.TestCheckResourceAttr(ResourceName+".test", "attribute_status.description", "absent"))},
		{Config: cleared, PlanOnly: true},
		{Config: observedOnly, Check: resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.displayname.*", "dW5vd25lZCBzZXJ2ZXIgdmFsdWU=")},
		{Config: observedOnly, PreConfig: func() {
			directory.mu.Lock()
			defer directory.mu.Unlock()
			directory.object.Attributes["othertelephone"] = [][]byte{[]byte("keep"), []byte("old"), []byte("unowned drift")}
		}, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "needs_update", "false"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", "bmV3"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", "dW5vd25lZCBkcmlmdA=="))},
		{Config: observedOnly, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for operation, expected := range map[string]int{"create": 1, "rename": 1, "move": 1, "update": 3, "delete": 1} {
		if directory.calls[operation] != expected {
			t.Errorf("%s calls=%d expected=%d", operation, directory.calls[operation], expected)
		}
	}
	if directory.exists {
		t.Fatal("leaf destruction did not retire the owned object")
	}
}
