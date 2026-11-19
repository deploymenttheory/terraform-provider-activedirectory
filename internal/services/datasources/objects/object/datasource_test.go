// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

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

// TestTerraformUnitObjectLookup preserves raw binary, hidden and ranged observations through GUID and DN reads.
func TestTerraformUnitObjectLookup(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	configuration, err := os.ReadFile("tests/terraform/unit/data-source.tf")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("tests/responses/object.json")
	if err != nil {
		t.Fatal(err)
	}
	object, err := mocks.DecodeObjectFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	other, err := mocks.DecodeObjectFixture([]byte(strings.ReplaceAll(string(payload), "dc93656c", "ec93656c")))
	if err != nil {
		t.Fatal(err)
	}
	const guid = "dc93656c-f745-4926-9bbb-31e7bbecb68b"
	guidConfig := string(configuration)
	dnConfig := strings.Replace(guidConfig, `object_guid = "`+guid+`"`, `dn = "`+object.DistinguishedName+`"`, 1)
	var seenGUID, seenDN atomic.Bool
	var mode atomic.Int32
	service := &mocks.Service{Read: func(_ context.Context, id shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
		if !slices.Contains(options.Properties, "jpegphoto") || !slices.Contains(options.Properties, "objectclass") || slices.Contains(options.Properties, "*") {
			t.Error("lookup did not preserve explicit raw and identity property selection")
		}
		switch id.Form() {
		case shared.IdentityFormGUID:
			seenGUID.Store(true)
		case shared.IdentityFormDN:
			seenDN.Store(true)
		default:
			t.Error("lookup inferred an unsupported identity form")
		}
		switch mode.Load() {
		case 1:
			return shared.IdentityResult{Object: &other, Complete: true}, nil
		case 2:
			return shared.IdentityResult{Object: &object}, shared.ErrIncompleteResult
		default:
			return shared.IdentityResult{Object: &object, Complete: true}, nil
		}
	}}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: guidConfig, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "id", guid),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "object_class", "contact"),
			resource.TestCheckTypeSetElemAttr("data."+DataSourceName+".test", "observed_attributes.jpegphoto.*", "/wCA"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "attribute_status.telephonenumber", "absent"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "attribute_status.member", "partial"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "attribute_ranges.%", "1"))},
		{Config: dnConfig, Check: resource.TestCheckResourceAttr("data."+DataSourceName+".test", "distinguished_name", object.DistinguishedName)},
		{Config: guidConfig, PreConfig: func() { mode.Store(1) }, ExpectError: regexp.MustCompile("Unexpected object lookup identity")},
		{Config: guidConfig, PreConfig: func() { mode.Store(2) }, ExpectError: regexp.MustCompile("Object lookup incomplete")},
	}})
	if !seenGUID.Load() || !seenDN.Load() {
		t.Fatal("both explicit object identity forms were not exercised")
	}
}
