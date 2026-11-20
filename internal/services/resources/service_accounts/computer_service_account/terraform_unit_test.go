// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account/mocks"
)

var errUnmanagedLink = errors.New("one-edge deletion changed an endpoint or unmanaged association")

const (
	computerID = "11111111-1111-4111-8111-111111111111"
	accountID  = "22222222-2222-4222-8222-222222222222"
	otherID    = "33333333-3333-4333-8333-333333333333"
)

func unitDirectory(t *testing.T) *mocks.Directory {
	t.Helper()
	data, err := os.ReadFile("tests/responses/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := mocks.DecodeDirectory(data)
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

// TestTerraformUnitSoleLinkDriftRepairsWithoutFalseAbsence keeps indeterminate identity and plans an explicit ensure operation for an empty attribute.
func TestTerraformUnitSoleLinkDriftRepairsWithoutFalseAbsence(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := unitDirectory(t)
	config, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory),
		CheckDestroy: func(*terraform.State) error {
			if directory.HasLink(computerID, accountID) || len(directory.Objects) != 3 {
				return errUnmanagedLink
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: string(config), Check: resource.TestCheckResourceAttr(ResourceName+".test", "is_associated", "true")},
			{
				Config: string(config), PreConfig: func() { directory.SetLink(computerID, accountID, false) },
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(ResourceName+".test", plancheck.ResourceActionUpdate),
				}}, Check: resource.TestCheckResourceAttr(ResourceName+".test", "is_associated", "true"),
			},
			{Config: string(config), PlanOnly: true},
		},
	})
	if directory.Adds != 2 || directory.Removes != 1 {
		t.Fatal("sole-link repair retried a mutation or failed to delete its own last link", directory.Adds, directory.Removes)
	}
}

// TestTerraformUnitDirectLinkLifecycle proves adoption, import, drift repair and one-edge deletion preserve unmanaged association.
func TestTerraformUnitDirectLinkLifecycle(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		t.Run(fmt.Sprint("adopted=", adopted), func(t *testing.T) {
			acceptance.IsolateUnitEnvironment(t)
			directory := unitDirectory(t)
			config, err := os.ReadFile("tests/terraform/unit/resource.tf")
			if err != nil {
				t.Fatal(err)
			}
			directory.SetLink(computerID, otherID, true)
			directory.SetLink(computerID, accountID, adopted)
			t.Chdir(t.TempDir())
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory),
				CheckDestroy: func(*terraform.State) error {
					if directory.HasLink(computerID, accountID) || !directory.HasLink(computerID, otherID) || len(directory.Objects) != 3 {
						return errUnmanagedLink
					}
					return nil
				},
				Steps: []resource.TestStep{
					{Config: string(config), Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(ResourceName+".test", "id", computerID+"/"+accountID),
						resource.TestCheckResourceAttr(ResourceName+".test", "service_account_dn", "CN=account,DC=unit,DC=invalid"),
					)},
					{Config: string(config), PlanOnly: true},
					{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
					{ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true},
					{Config: string(config), PreConfig: func() { directory.SetLink(computerID, accountID, false) }},
					{Config: string(config), PlanOnly: true},
				},
			})
			expected := 2
			if adopted {
				expected = 1
			}
			if directory.Adds != expected || directory.Removes != 1 {
				t.Fatal("unexpected direct-link write count", directory.Adds, directory.Removes)
			}
		})
	}
}
