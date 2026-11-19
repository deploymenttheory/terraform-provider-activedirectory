// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	managed "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/service_account"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/service_account/mocks"
)

func TestTerraformUnitQueryIdentityAndResourceSnapshot(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/list-resource.tfquery.hcl")
	if err != nil {
		t.Fatal(err)
	}
	providers, err := os.ReadFile("tests/terraform/unit/providers.tf")
	if err != nil {
		t.Fatal(err)
	}
	unit := fixtureUnit(t)
	service := &mocks.Service{List: func(_ context.Context, options shared.SearchOptions) (service_accounts.ServiceAccountsResult, error) {
		if options.Base != "DC=unit,DC=invalid" || options.PageSize != 2 || options.ResultLimit != 10 {
			t.Error("query ignored its selected bounds")
		}
		return service_accounts.ServiceAccountsResult{Accounts: []service_accounts.ServiceAccount{unit}, Observation: shared.SearchResult{Complete: true}}, nil
	}}
	t.Chdir(t.TempDir())
	testingresource.Test(t, testingresource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []testingresource.TestStep{
		{Config: string(providers)},
		{Query: true, Config: string(config), QueryResultChecks: []querycheck.QueryResultCheck{
			querycheck.ExpectLength(ListResourceName+".test", 1),
			querycheck.ExpectIdentity(ListResourceName+".test", map[string]knownvalue.Check{"id": knownvalue.StringExact(unit.GUID.String())}),
			querycheck.ExpectResourceKnownValues(ListResourceName+".test", nil, []querycheck.KnownValueCheck{
				{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact("fixture")},
				{Path: tfjsonpath.New("parent_dn"), KnownValue: knownvalue.StringExact("DC=unit,DC=invalid")},
				{Path: tfjsonpath.New("enabled"), KnownValue: knownvalue.Bool(false)},
			}),
		}},
	}})
}

func TestStreamingStopsAndPreservesPartialFailure(t *testing.T) {
	unit := fixtureUnit(t)
	reads := 0
	incomplete := false
	service := &mocks.Service{List: func(context.Context, shared.SearchOptions) (service_accounts.ServiceAccountsResult, error) {
		reads++
		result := service_accounts.ServiceAccountsResult{
			Accounts:    []service_accounts.ServiceAccount{unit, unit},
			Observation: shared.SearchResult{Complete: !incomplete},
		}
		if incomplete {
			result.Accounts = result.Accounts[:1]
			return result, shared.ErrIncompleteResult
		}
		return result, nil
	}}
	r := &ServiceAccountListResource{client: service}
	request := listRequest(t, r)
	var stream list.ListResultsStream
	r.List(t.Context(), request, &stream)
	if reads != 0 {
		t.Fatal("SDK query ran before the consumer started iteration")
	}
	emitted := 0
	stream.Results(func(result list.ListResult) bool { emitted++; return false })
	if reads != 1 || emitted != 1 {
		t.Fatal("consumer stop did not halt result emission")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reads = 0
	r.List(ctx, request, &stream)
	stream.Results(func(result list.ListResult) bool {
		if !result.Diagnostics.HasError() {
			t.Error("cancellation not diagnosed")
		}
		return true
	})
	if reads != 0 {
		t.Fatal("canceled discovery contacted the directory")
	}
	incomplete = true
	r.List(t.Context(), request, &stream)
	emitted, failures := 0, 0
	stream.Results(func(result list.ListResult) bool {
		if result.Diagnostics.HasError() {
			failures++
		} else {
			emitted++
		}
		return true
	})
	if emitted != 1 || failures != 1 {
		t.Fatal("partial read failure discarded a usable identity or claimed completion")
	}
	incomplete = false
	r.List(t.Context(), request, &stream)
	emitted, failures = 0, 0
	stream.Results(func(result list.ListResult) bool {
		if result.Diagnostics.HasError() {
			failures++
		} else {
			emitted++
		}
		return true
	})
	if emitted != 1 || failures != 1 {
		t.Fatal("duplicate import identities were emitted without a diagnostic")
	}
}

func fixtureUnit(t *testing.T) service_accounts.ServiceAccount {
	t.Helper()
	payload, err := os.ReadFile("tests/responses/service_account.json")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := mocks.DecodeServiceAccountFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	return unit
}

func listRequest(t *testing.T, r *ServiceAccountListResource) list.ListRequest {
	t.Helper()
	var config list.ListResourceSchemaResponse
	r.ListResourceConfigSchema(t.Context(), list.ListResourceSchemaRequest{}, &config)
	plan := tfsdk.Plan{Schema: config.Schema}
	model := ServiceAccountListModel{
		BaseDN: types.StringValue("DC=unit,DC=invalid"), Scope: types.StringValue("one_level"),
		LDAPFilter: types.StringNull(), PageSize: types.Int64Value(2), ResultLimit: types.Int64Value(10),
	}
	if diagnostics := plan.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	managedResource := managed.NewServiceAccountResource().(*managed.ServiceAccountResource)
	var resourceSchema resource.SchemaResponse
	managedResource.Schema(t.Context(), resource.SchemaRequest{}, &resourceSchema)
	var identitySchema resource.IdentitySchemaResponse
	managedResource.IdentitySchema(t.Context(), resource.IdentitySchemaRequest{}, &identitySchema)
	return list.ListRequest{
		Config: tfsdk.Config{Schema: config.Schema, Raw: plan.Raw}, ResourceSchema: resourceSchema.Schema,
		ResourceIdentitySchema: identitySchema.IdentitySchema, IncludeResource: true, Limit: 10,
	}
}
