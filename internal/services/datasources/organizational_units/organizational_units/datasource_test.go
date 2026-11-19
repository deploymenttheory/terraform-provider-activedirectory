// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

func TestTerraformUnitBoundedSortedDiscoveryAndIncompleteSearch(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("tests/responses/organizational_unit.json")
	if err != nil {
		t.Fatal(err)
	}
	first, err := mocks.DecodeUnitFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := mocks.DecodeUnitFixture([]byte(strings.ReplaceAll(string(payload), "a3e9767a", "b3e9767a")))
	if err != nil {
		t.Fatal(err)
	}
	var incomplete atomic.Bool
	service := &mocks.Service{List: func(_ context.Context, options shared.SearchOptions) (organizational_units.OrganizationalUnitsResult, error) {
		if options.Base != "DC=unit,DC=invalid" || options.Scope != shared.ScopeOneLevel || options.PageSize != 2 || options.ResultLimit != 10 {
			t.Error("discovery changed the explicit query boundary")
		}
		result := organizational_units.OrganizationalUnitsResult{
			Units:       []organizational_units.OrganizationalUnit{second, first},
			Observation: shared.SearchResult{Complete: !incomplete.Load()},
		}
		if incomplete.Load() {
			return result, shared.ErrIncompleteResult
		}
		return result, nil
	}}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: string(config), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "organizational_units.#", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "organizational_units.0.id", first.GUID.String()),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "organizational_units.1.id", second.GUID.String()),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "limit_reached", "false"))},
		{Config: string(config), PreConfig: func() { incomplete.Store(true) }, ExpectError: regexp.MustCompile("OU discovery incomplete")},
	}})
}
