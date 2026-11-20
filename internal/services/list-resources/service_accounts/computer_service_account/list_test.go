// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"
	"fmt"
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

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
	managed "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/computer_service_account/mocks"
)

const (
	computerID = "11111111-1111-4111-8111-111111111111"
	accountID  = "22222222-2222-4222-8222-222222222222"
)

func unitDirectory(t *testing.T) *mocks.Directory {
	t.Helper()
	payload, err := os.ReadFile("tests/responses/objects.json")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := mocks.DecodeDirectory(payload)
	if err != nil {
		t.Fatal(err)
	}
	directory.SetLink(computerID, accountID, true)
	return directory
}

// TestTerraformUnitCompositeImportQuery proves list discovery emits the matching resource identity and complete endpoint snapshot.
func TestTerraformUnitCompositeImportQuery(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	directory := unitDirectory(t)
	providers, err := os.ReadFile("tests/terraform/unit/providers.tf")
	if err != nil {
		t.Fatal(err)
	}
	query, err := os.ReadFile("tests/terraform/unit/list-resource.tfquery.hcl")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	testingresource.Test(t, testingresource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory), Steps: []testingresource.TestStep{
		{Config: string(providers)},
		{Query: true, Config: string(query), QueryResultChecks: []querycheck.QueryResultCheck{
			querycheck.ExpectLength(ListResourceName+".test", 1),
			querycheck.ExpectIdentity(ListResourceName+".test", map[string]knownvalue.Check{"id": knownvalue.StringExact(computerID + "/" + accountID)}),
			querycheck.ExpectResourceKnownValues(ListResourceName+".test", nil, []querycheck.KnownValueCheck{
				{Path: tfjsonpath.New("computer_guid"), KnownValue: knownvalue.StringExact(computerID)},
				{Path: tfjsonpath.New("service_account_guid"), KnownValue: knownvalue.StringExact(accountID)},
				{Path: tfjsonpath.New("service_account_dn"), KnownValue: knownvalue.StringExact("CN=account,DC=unit,DC=invalid")},
			}),
		}},
	}})
}

// TestQueryLazinessCancellationAndPartialFailure checks the SDK call boundary and retention of usable imports before an error.
func TestQueryLazinessCancellationAndPartialFailure(t *testing.T) {
	directory := unitDirectory(t)
	reads, partial := 0, false
	service := &mocks.Service{
		Resolve: directory.ResolveIdentity,
		Assert:  directory.Search,
		Links: func(ctx context.Context, target shared.Identity, options sdkaccounts.ComputerServiceAccountsOptions) (sdkaccounts.ComputerServiceAccountsResult, error) {
			reads++
			result, err := directory.ComputerServiceAccounts(ctx, target, options)
			if partial {
				result.Complete = false
				return result, shared.ErrIncompleteResult
			}
			if err != nil {
				return result, fmt.Errorf("read synthetic account links: %w", err)
			}
			return result, nil
		},
	}
	r := &ComputerServiceAccountListResource{client: service}
	req := listRequest(t, r)
	var stream list.ListResultsStream
	r.List(t.Context(), req, &stream)
	if reads != 0 {
		t.Fatal("query ran before iteration")
	}
	rows := 0
	stream.Results(func(list.ListResult) bool { rows++; return false })
	if reads != 1 || rows != 1 {
		t.Fatal("consumer stop was ignored")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r.List(ctx, req, &stream)
	stream.Results(func(result list.ListResult) bool {
		if !result.Diagnostics.HasError() {
			t.Error("canceled query did not report cancellation")
		}
		return true
	})
	if reads != 1 {
		t.Fatal("canceled query performed SDK I/O")
	}
	partial, rows = true, 0
	failures := 0
	r.List(t.Context(), req, &stream)
	stream.Results(func(result list.ListResult) bool {
		if result.Diagnostics.HasError() {
			failures++
		} else {
			rows++
		}
		return true
	})
	if rows != 1 || failures != 1 {
		t.Fatal("partial discovery discarded a usable link or claimed completion", rows, failures)
	}
}

func listRequest(t *testing.T, r *ComputerServiceAccountListResource) list.ListRequest {
	t.Helper()
	var config list.ListResourceSchemaResponse
	r.ListResourceConfigSchema(t.Context(), list.ListResourceSchemaRequest{}, &config)
	plan := tfsdk.Plan{Schema: config.Schema}
	model := common.QueryModel{ComputerGUID: types.StringValue(computerID), ResultLimit: types.Int64Value(10)}
	if diagnostics := plan.Set(t.Context(), model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	managedResource := &managed.ComputerServiceAccountResource{}
	var resourceSchema resource.SchemaResponse
	managedResource.Schema(t.Context(), resource.SchemaRequest{}, &resourceSchema)
	var identitySchema resource.IdentitySchemaResponse
	managedResource.IdentitySchema(t.Context(), resource.IdentitySchemaRequest{}, &identitySchema)
	return list.ListRequest{
		Config: tfsdk.Config{Schema: config.Schema, Raw: plan.Raw}, ResourceSchema: resourceSchema.Schema,
		ResourceIdentitySchema: identitySchema.IdentitySchema, IncludeResource: true, Limit: 10,
	}
}
