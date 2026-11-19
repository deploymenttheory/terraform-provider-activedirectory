// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

const (
	firstSecret  = "synthetic-ephemeral-computer-secret-first"
	secondSecret = "synthetic-ephemeral-computer-secret-second"
)

// secretFreePlan inspects Terraform's complete JSON plan rather than trusting a single selected attribute.
type secretFreePlan struct{}

func (secretFreePlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	if strings.Contains(string(encoded), firstSecret) || strings.Contains(string(encoded), secondSecret) {
		resp.Error = errSecretInArtifact
	}
}

func TestTerraformUnitComputerLifecycleAndWriteOnlyPassword(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := strings.ReplaceAll(string(data), "\r\n", "\n")
	updated := strings.NewReplacer(
		`"initial"`, `"renamed"`, `OU=first,DC=unit,DC=invalid`, `OU=second,DC=unit,DC=invalid`,
		"first description", "second description", "password_wo_version = 1", "password_wo_version = 2").
		Replace(initial)
	unmanaged := strings.Replace(updated, "  description         = \"second description\"\n", "", 1)
	cleared := strings.Replace(unmanaged, "  enabled", "  clear_properties = [\"description\"]\n  enabled", 1)
	directory := newUnitDirectory(t)
	t.Chdir(t.TempDir())
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretFreePlan{}}}
	checkState := func(state *terraform.State) error {
		for _, module := range state.Modules {
			for _, object := range module.Resources {
				if object.Primary != nil {
					for _, value := range object.Primary.Attributes {
						if strings.Contains(value, firstSecret) || strings.Contains(value, secondSecret) {
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
			Config:           initial,
			ConfigPlanChecks: checks,
			Check: resource.ComposeAggregateTestCheckFunc(
				checkState,
				resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
				resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "1"),
				resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "true"),
			),
		},
		{Config: initial, PlanOnly: true},
		{
			Config:           updated,
			PreConfig:        func() { t.Setenv("TF_VAR_computer_password", secondSecret) },
			ConfigPlanChecks: checks,
			Check:            resource.ComposeAggregateTestCheckFunc(checkState, resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "2")),
		},
		{Config: unmanaged, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{
			ResourceName:    ResourceName + ".test",
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithResourceIdentity,
			GenerateConfig:  true,
			ImportPlanChecks: resource.ImportPlanChecks{
				PreApply: []plancheck.PlanCheck{secretFreePlan{}, plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)},
			},
		},
		{
			Config:           cleared,
			ConfigPlanChecks: checks,
			Check:            resource.ComposeAggregateTestCheckFunc(checkState, resource.TestCheckNoResourceAttr(ResourceName+".test", "description")),
		},
		{Config: cleared, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for operation, expected := range map[string]int{"create": 1, "rename": 1, "move": 1, "password": 2, "update": 2, "delete": 1} {
		if directory.calls[operation] != expected {
			t.Errorf("%s calls=%d expected=%d", operation, directory.calls[operation], expected)
		}
	}
	for _, buffer := range directory.borrowedPasswords {
		for _, value := range buffer {
			if value != 0 {
				t.Fatal("caller-owned password buffer was retained after SDK use")
			}
		}
	}
}

func TestTerraformUnitAcceptedCreationSurvivesEnableFailure(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	create := service.Create
	service.Create = func(ctx context.Context, request computers.CreateComputerRequest) (computers.ComputerCreationResult, error) {
		result, createErr := create(ctx, request)
		if createErr != nil {
			return result, createErr
		}
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.model.Enabled = types.BoolValue(false)
		result.Computer = directory.snapshot(t)
		result.Complete = false
		result.Enable = &shared.ObjectMutationResult{State: shared.ObjectWriteRejected, TargetGUID: result.Creation.TargetGUID}
		return result, errors.Join(computers.ErrComputerCreationIncomplete, shared.ErrObjectWritePermission)
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: string(data), ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: string(data), ExpectError: regexp.MustCompile("Computer mutation still unresolved")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{Config: string(data), Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["create"] != 1 || directory.calls["password"] != 1 || directory.calls["delete"] != 0 {
		t.Fatal("accepted creation/password was replaced or replayed after enable failure")
	}
}

func TestTerraformUnitUncertainPasswordCannotReplay(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_computer_password", firstSecret)
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := string(data)
	updated := strings.Replace(initial, "password_wo_version = 1", "password_wo_version = 2", 1)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	service.Password = func(_ context.Context, request accounts.PasswordResetRequest) (accounts.PasswordResult, error) {
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.calls["password"]++
		directory.borrowedPasswords = append(directory.borrowedPasswords, request.NewPassword)
		guid, _ := shared.ParseGUID(fixtureGUID)
		return accounts.PasswordResult{State: shared.ObjectWriteUncertain, TargetGUID: &guid}, accounts.ErrPasswordUncertain
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: initial},
		{Config: updated, ExpectError: regexp.MustCompile("Computer password change did not complete")},
		{Config: updated, ExpectError: regexp.MustCompile("Computer mutation still unresolved")},
		{Config: updated, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["password"] != 2 || directory.calls["delete"] != 0 {
		t.Fatal("uncertain password write replayed or authorized deletion")
	}
}
