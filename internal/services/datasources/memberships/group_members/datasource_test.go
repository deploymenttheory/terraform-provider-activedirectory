// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

// TestTerraformUnitDirectAndRecursive performs real Terraform refreshes through the isolated read-only service boundary.
func TestTerraformUnitDirectAndRecursive(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	f := loadFixture(t)
	config := fixtureConfig(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, f.service("")), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.#", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.0.id", userID),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.1.id", nestedID),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.0.primary", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"),
		)},
		{Config: regexp.MustCompile(`recursive\s*=\s*false`).ReplaceAllString(config, "recursive = true"), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.#", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.1.id", computerID),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.0.primary", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "groups_visited", "2"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "true"),
		)},
	}})
}

// TestTerraformUnitPrimaryHiddenAndBounds preserves primary-only rows and never claims hidden or budget-stopped collections are complete.
func TestTerraformUnitPrimaryHiddenAndBounds(t *testing.T) {
	for _, mode := range []string{"primary", "hidden", "bounded"} {
		t.Run(mode, func(t *testing.T) {
			acceptance.IsolateUnitEnvironment(t)
			f := loadFixture(t)
			config := fixtureConfig(t)
			t.Chdir(t.TempDir())
			count := "1"
			if mode == "hidden" {
				count = "0"
			}
			resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, f.service(mode)), Steps: []resource.TestStep{
				{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data."+DataSourceName+".test", "members.#", count),
					resource.TestCheckResourceAttr("data."+DataSourceName+".test", "complete", "false"),
				)},
			}})
		})
	}
}

// TestQueryConstructionRefusesUnknownOrUnboundedInputs rejects malformed scope and every invalid budget without SDK calls.
func TestQueryConstructionRefusesUnknownOrUnboundedInputs(t *testing.T) {
	for _, value := range []types.Int64{types.Int64Unknown(), types.Int64Value(0), types.Int64Value(-1), types.Int64Value(10001)} {
		model := GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID), MaxObjects: value}
		if _, _, diagnostics := constructQuery(model); !diagnostics.HasError() {
			t.Fatal("invalid object budget was accepted", value)
		}
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("CN=root,DC=unit"), types.StringValue("00000000-0000-0000-0000-000000000000")} {
		if _, _, diagnostics := constructQuery(GroupMembersDataSourceModel{GroupGUID: value}); !diagnostics.HasError() {
			t.Fatal("invalid query GUID was accepted", value)
		}
	}
	if _, _, diagnostics := constructQuery(GroupMembersDataSourceModel{GroupGUID: types.StringValue(rootID), Recursive: types.BoolUnknown()}); !diagnostics.HasError() {
		t.Fatal("unknown recursive traversal was accepted")
	}
}

// TestMixedBoundAndPrivateFailureIsNotDowngradedToWarning exercises the real Terraform diagnostic path for a partial failed query.
func TestMixedBoundAndPrivateFailureIsNotDowngradedToWarning(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	f := loadFixture(t)
	service := f.service("bounded")
	private := errPrivatePeer
	service.Query = func(context.Context, shared.Identity, memberships.GroupMembersOptions) (memberships.GroupMembersResult, error) {
		return f.result("bounded"), errors.Join(memberships.ErrMembershipLimit, private)
	}
	config := fixtureConfig(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Group membership discovery incomplete")},
	}})
	if onlyLimits(fmt.Errorf("scope: %w", errors.Join(memberships.ErrMembershipLimit, private))) {
		t.Fatal("mixed private failure became a harmless bound")
	}
}

// TestContradictoryRootStopsBeforeTraversal refuses a different positive scope rather than issuing a query for it.
func TestContradictoryRootStopsBeforeTraversal(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	f := loadFixture(t)
	group, _ := shared.IdentityByGUID(rootID)
	wrong := f.objects[nestedID]
	if _, valid := groupMetadata(shared.IdentityResult{Object: &wrong, Complete: true}, group); valid {
		t.Fatal("a different group GUID became this query's scope")
	}
	service := f.service("")
	service.Resolve = func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
		return shared.IdentityResult{Object: &wrong, Candidates: []shared.Object{wrong}, Complete: true}, nil
	}
	config := fixtureConfig(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Group membership scope unavailable")},
	}})
	if f.queries != 0 {
		t.Fatal("a contradictory scope allowed a membership query")
	}
}
