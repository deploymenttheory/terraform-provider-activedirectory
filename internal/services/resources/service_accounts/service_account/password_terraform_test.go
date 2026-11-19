// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

const creationSecret = "unit-only-ephemeral-service-account-secret"

var errSecretInArtifact = errors.New("secret found in artifact or artifact inspection failed")

type secretFreePlan struct{}

func (secretFreePlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil || strings.Contains(string(encoded), creationSecret) {
		resp.Error = errSecretInArtifact
	}
}

// TestTerraformUnitStandalonePasswordNeverEntersArtifacts verifies write-only creation input, replacement planning and caller-buffer erasure.
func TestTerraformUnitStandalonePasswordNeverEntersArtifacts(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_service_account_password", creationSecret)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	config := strings.ReplaceAll(string(data), "\r\n", "\n")
	config = strings.Replace(config, `kind                    = "group_managed"`, `kind                    = "standalone"`, 1)
	config = strings.Replace(config, "  password_reader_sids    = [\"S-1-5-21-1-2-3-1001\"]\n", "", 1)
	config = strings.Replace(
		config,
		"  enabled                 = true",
		"  password_wo = var.service_account_password\n  password_wo_version = 1\n  enabled                 = true",
		1,
	)
	config += "\nvariable \"service_account_password\" {\n type = string\n sensitive = true\n ephemeral = true\n}\n"
	replaced := strings.Replace(config, "password_wo_version = 1", "password_wo_version = 2", 1)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	checkState := func(state *terraform.State) error {
		for _, module := range state.Modules {
			for _, object := range module.Resources {
				if object.Primary != nil {
					for _, value := range object.Primary.Attributes {
						if strings.Contains(value, creationSecret) {
							return errSecretInArtifact
						}
					}
				}
			}
		}
		return nil
	}
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{
			Config: config, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}},
			Check: resource.ComposeAggregateTestCheckFunc(checkState, resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "1")),
		},
		{Config: config, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{
			Config: replaced,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{secretFreePlan{}, plancheck.ExpectResourceAction(ResourceName+".test", plancheck.ResourceActionDestroyBeforeCreate)},
			},
		},
		{Config: replaced, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["create"] != 2 || directory.calls["password"] != 2 || directory.guid == fixtureGUID {
		t.Fatal("a plan, import or refresh replayed creation/password setup")
	}
	for _, buffer := range directory.borrowedPasswords {
		for _, value := range buffer {
			if value != 0 {
				t.Fatal("caller-owned password buffer was not cleared")
			}
		}
	}
}
