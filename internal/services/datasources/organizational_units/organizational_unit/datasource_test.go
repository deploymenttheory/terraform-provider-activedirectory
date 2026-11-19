// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

func TestTerraformUnitLookupByGUIDAndDN(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("tests/responses/organizational_unit.json")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := mocks.DecodeUnitFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	guidConfig := string(config)
	dnConfig := strings.Replace(guidConfig, `object_guid = "`+unit.GUID.String()+`"`, `dn = "`+unit.DistinguishedName+`"`, 1)
	var seenGUID, seenDN atomic.Bool
	service := &mocks.Service{Read: func(_ context.Context, id shared.Identity, options shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
		if options.Target != shared.IdentityTargetObject {
			t.Error("lookup broadened its allowed identity target")
		}
		switch id.Form() {
		case shared.IdentityFormGUID:
			seenGUID.Store(true)
		case shared.IdentityFormDN:
			seenDN.Store(true)
		default:
			t.Error("lookup guessed an unsupported identity form")
		}
		return organizational_units.OrganizationalUnitResult{Unit: &unit, Observation: shared.IdentityResult{Complete: true, Object: &unit.Raw}}, nil
	}}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: guidConfig, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "id", unit.GUID.String()),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "description", "fixture description"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "protected_from_accidental_deletion", "false"),
			resource.TestCheckNoResourceAttr("data."+DataSourceName+".test", "city"))},
		{Config: dnConfig, Check: resource.TestCheckResourceAttr("data."+DataSourceName+".test", "distinguished_name", unit.DistinguishedName)},
	}})
	if !seenGUID.Load() || !seenDN.Load() {
		t.Error("Terraform did not exercise both explicit identity forms")
	}
}
