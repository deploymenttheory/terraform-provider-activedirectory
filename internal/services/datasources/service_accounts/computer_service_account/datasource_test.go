// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account/mocks"
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
	computer := "11111111-1111-4111-8111-111111111111"
	account := "22222222-2222-4222-8222-222222222222"
	other := "33333333-3333-4333-8333-333333333333"
	directory.SetLink(computer, other, true)
	directory.SetLink(computer, account, true)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory), Steps: []resource.TestStep{
		{Config: string(config), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "id", computer+"/"+account),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "computer_dn", "CN=computer,DC=unit,DC=invalid"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "service_account_dn", "CN=account,DC=unit,DC=invalid"),
		)},
	}})
	if directory.Adds != 0 || directory.Removes != 0 {
		t.Fatal("read-only data source submitted a association mutation")
	}
}
