// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	managed "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

// TestTerraformUnitObjectQueryUsesActualResourceSchema verifies GUID identity and snapshots against the implemented managed resource.
func TestTerraformUnitObjectQueryUsesActualResourceSchema(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	configuration, err := os.ReadFile("tests/terraform/unit/list-resource.tfquery.hcl")
	if err != nil {
		t.Fatal(err)
	}
	providers, err := os.ReadFile("tests/terraform/unit/providers.tf")
	if err != nil {
		t.Fatal(err)
	}
	object := fixtureObject(t)
	service := &mocks.Service{List: func(_ context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
		if options.Base != "DC=unit,DC=invalid" || options.PageSize != 2 || options.ResultLimit != 10 {
			t.Error("generic query ignored explicit bounds")
		}
		return shared.SearchResult{Complete: true, Objects: []shared.Object{object}}, nil
	}}
	t.Chdir(t.TempDir())
	testingresource.Test(t, testingresource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []testingresource.TestStep{
		{Config: string(providers)},
		{Query: true, Config: string(configuration), QueryResultChecks: []querycheck.QueryResultCheck{
			querycheck.ExpectLength(ListResourceName+".test", 1),
			querycheck.ExpectIdentity(ListResourceName+".test", map[string]knownvalue.Check{"id": knownvalue.StringExact(fixtureGUID)}),
			querycheck.ExpectResourceKnownValues(ListResourceName+".test", nil, []querycheck.KnownValueCheck{
				{Path: tfjsonpath.New("name"), KnownValue: knownvalue.StringExact("Fixture contact")},
				{Path: tfjsonpath.New("object_class"), KnownValue: knownvalue.StringExact("contact")},
				{Path: tfjsonpath.New("parent_dn"), KnownValue: knownvalue.StringExact("OU=owned,DC=unit,DC=invalid")},
				{Path: tfjsonpath.New("needs_update"), KnownValue: knownvalue.Bool(false)},
			}),
		}},
	}})
}

// TestObjectStreamPreservesPartialFailureAndStops verifies lazy bounded materialization, consumer stop, cancellation and duplicate refusal.
func TestObjectStreamPreservesPartialFailureAndStops(t *testing.T) {
	object := fixtureObject(t)
	reads, mode := 0, "duplicate"
	service := &mocks.Service{List: func(_ context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
		reads++
		if mode == "bounded" {
			if options.ResultLimit != 1 {
				t.Error("Terraform's smaller limit was not forwarded to SDK materialization")
			}
			return shared.SearchResult{Objects: []shared.Object{object}, LimitReached: true}, nil
		}
		if mode == "partial" {
			return shared.SearchResult{Objects: []shared.Object{object}}, shared.ErrIncompleteResult
		}
		return shared.SearchResult{Objects: []shared.Object{object, object}, Complete: true}, nil
	}}
	r := &ObjectListResource{client: service}
	req := listRequest(t, r)
	var stream list.ListResultsStream
	r.List(t.Context(), req, &stream)
	if reads != 0 {
		t.Fatal("SDK search ran before iteration began")
	}
	rows := 0
	stream.Results(func(list.ListResult) bool { rows++; return false })
	if reads != 1 || rows != 1 {
		t.Fatal("consumer stop did not halt emission")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reads = 0
	r.List(ctx, req, &stream)
	stream.Results(func(row list.ListResult) bool {
		if !row.Diagnostics.HasError() {
			t.Error("cancellation was not diagnosed")
		}
		return true
	})
	if reads != 0 {
		t.Fatal("canceled query contacted the SDK")
	}
	for _, scenario := range []string{"partial", "duplicate", "bounded"} {
		mode = scenario
		request := req
		if scenario == "bounded" {
			request.Limit = 1
		}
		r.List(t.Context(), request, &stream)
		rows, errors, warnings := 0, 0, 0
		stream.Results(func(row list.ListResult) bool {
			if row.DisplayName != "" {
				rows++
			}
			if row.Diagnostics.HasError() {
				errors++
			}
			for _, diagnostic := range row.Diagnostics {
				if diagnostic.Severity() == diag.SeverityWarning {
					warnings++
				}
			}
			return true
		})
		if rows != 1 || scenario != "bounded" && errors != 1 || scenario == "bounded" && (errors != 0 || warnings != 1) {
			t.Fatal("stream lost a usable identity, repeated a GUID or misrepresented bounded/partial completion")
		}
	}
}

// fixtureObject reads an explicit public SDK observation with raw binary attributes.
func fixtureObject(t *testing.T) shared.Object {
	t.Helper()
	payload, err := os.ReadFile("tests/responses/object.json")
	if err != nil {
		t.Fatal(err)
	}
	object, err := mocks.DecodeObjectFixture(payload)
	if err != nil {
		t.Fatal(err)
	}
	return object
}

// listRequest uses the actual managed resource schema and GUID identity, never a test-only resource stub.
func listRequest(t *testing.T, r *ObjectListResource) list.ListRequest {
	t.Helper()
	var config list.ListResourceSchemaResponse
	r.ListResourceConfigSchema(t.Context(), list.ListResourceSchemaRequest{}, &config)
	plan := tfsdk.Plan{Schema: config.Schema}
	model := ObjectListModel{
		BaseDN: types.StringValue("DC=unit,DC=invalid"), Scope: types.StringValue("one_level"), LDAPFilter: types.StringNull(),
		PageSize: types.Int64Value(2), ResultLimit: types.Int64Value(10),
		Properties: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("description"), types.StringValue("thumbnailPhoto")}),
	}
	if diagnostics := plan.Set(t.Context(), &model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	rsrc := managed.NewObjectResource().(*managed.ObjectResource)
	var resourceSchema resource.SchemaResponse
	var identitySchema resource.IdentitySchemaResponse
	rsrc.Schema(t.Context(), resource.SchemaRequest{}, &resourceSchema)
	rsrc.IdentitySchema(t.Context(), resource.IdentitySchemaRequest{}, &identitySchema)
	return list.ListRequest{
		Config: tfsdk.Config{Schema: config.Schema, Raw: plan.Raw}, ResourceSchema: resourceSchema.Schema,
		ResourceIdentitySchema: identitySchema.IdentitySchema, IncludeResource: true, Limit: 10,
	}
}
