// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"context"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

// TestTerraformUnitObjectDiscovery separates complete, caller-bounded, partial and duplicate-identity results.
func TestTerraformUnitObjectDiscovery(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	configuration, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("tests/responses/object.json")
	if err != nil {
		t.Fatal(err)
	}
	first, err := mocks.DecodeObjectFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := mocks.DecodeObjectFixture([]byte(strings.ReplaceAll(string(payload), "dc93656c", "ec93656c")))
	if err != nil {
		t.Fatal(err)
	}
	var mode atomic.Int32
	service := &mocks.Service{List: func(_ context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
		if options.Base != "DC=unit,DC=invalid" || options.Filter != "(objectClass=contact)" || options.Scope != shared.ScopeOneLevel || options.PageSize != 2 ||
			!slices.Contains(options.Properties, "thumbnailphoto") {
			t.Error("generic discovery changed the explicit search boundary or raw property selection")
		}
		switch mode.Load() {
		case 1:
			if options.ResultLimit != 1 {
				t.Error("configured discovery bound was not passed to the SDK")
			}
			return shared.SearchResult{Objects: []shared.Object{first}, LimitReached: true}, nil
		case 2:
			return shared.SearchResult{Objects: []shared.Object{first}}, shared.ErrIncompleteResult
		case 3:
			return shared.SearchResult{Objects: []shared.Object{first, first}, Complete: true}, nil
		default:
			return shared.SearchResult{Objects: []shared.Object{second, first}, Complete: true}, nil
		}
	}}
	config := string(configuration)
	bounded := strings.Replace(config, "result_limit = 10", "result_limit = 1", 1)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "objects.#", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "objects.0.id", "dc93656c-f745-4926-9bbb-31e7bbecb68b"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "objects.1.id", "ec93656c-f745-4926-9bbb-31e7bbecb68b"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"))},
		{Config: bounded, PreConfig: func() { mode.Store(1) }, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "objects.#", "1"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "false"))},
		{Config: config, PreConfig: func() { mode.Store(2) }, ExpectError: regexp.MustCompile("Object discovery incomplete")},
		{Config: config, PreConfig: func() { mode.Store(3) }, ExpectError: regexp.MustCompile("Duplicate object identity")},
	}})
}
