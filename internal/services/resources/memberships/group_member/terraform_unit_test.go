// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
)

var errUnmanagedLink = errors.New("one-edge deletion changed an endpoint or unmanaged membership")

const (
	groupID  = "11111111-1111-4111-8111-111111111111"
	memberID = "22222222-2222-4222-8222-222222222222"
	otherID  = "33333333-3333-4333-8333-333333333333"
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

// TestTerraformUnitDirectLinkLifecycle proves adoption, import, drift repair and one-edge deletion preserve unmanaged membership.
func TestTerraformUnitDirectLinkLifecycle(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		t.Run(fmt.Sprint("adopted=", adopted), func(t *testing.T) {
			acceptance.IsolateUnitEnvironment(t)
			directory := unitDirectory(t)
			config, err := os.ReadFile("tests/terraform/unit/resource.tf")
			if err != nil {
				t.Fatal(err)
			}
			directory.SetLink(groupID, otherID, true)
			directory.SetLink(groupID, memberID, adopted)
			t.Chdir(t.TempDir())
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory),
				CheckDestroy: func(*terraform.State) error {
					if directory.HasLink(groupID, memberID) || !directory.HasLink(groupID, otherID) || len(directory.Objects) != 3 {
						return errUnmanagedLink
					}
					return nil
				},
				Steps: []resource.TestStep{
					{Config: string(config), Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(ResourceName+".test", "id", groupID+"/"+memberID),
						resource.TestCheckResourceAttr(ResourceName+".test", "member_dn", "CN=member,DC=unit,DC=invalid"),
					)},
					{Config: string(config), PlanOnly: true},
					{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
					{ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true},
					{Config: string(config), PreConfig: func() { directory.SetLink(groupID, memberID, false) }},
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
