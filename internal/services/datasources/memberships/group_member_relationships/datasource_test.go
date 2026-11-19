// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member_relationships

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/memberships/group_member/mocks"
)

// TestTerraformUnitDirectLookup exercises this data source with its own feature-local fixture and isolated provider.
func TestTerraformUnitDirectLookup(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	payload, err := os.ReadFile("tests/responses/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := mocks.DecodeDirectory(payload)
	if err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	group := "11111111-1111-4111-8111-111111111111"
	member := "22222222-2222-4222-8222-222222222222"
	other := "33333333-3333-4333-8333-333333333333"
	directory.SetLink(group, other, true)
	directory.SetLink(group, member, true)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory), Steps: []resource.TestStep{
		{Config: string(config), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "id", group),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.#", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.0.id", group+"/"+member),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.1.id", group+"/"+other),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "limit_reached", "false"),
		)},
		{
			Config:    string(config),
			PreConfig: func() { directory.SetLink(group, member, false); directory.SetLink(group, other, false) },
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.#", "0"),
				resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"),
			),
		},
	}})
	if directory.Adds != 0 || directory.Removes != 0 {
		t.Fatal("read-only data source submitted a membership mutation")
	}
}
