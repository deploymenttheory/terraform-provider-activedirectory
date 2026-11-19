// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

// Registration fixtures implement metadata only; directory operations are never invoked by these tests.
type resourceFixture struct{ resource.Resource }

func (*resourceFixture) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fixture"
}

type dataSourceFixture struct{ datasource.DataSource }

func (*dataSourceFixture) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fixture"
}

type listFixture struct{ list.ListResource }

func (*listFixture) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fixture"
}

type actionFixture struct{ action.Action }

func (*actionFixture) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fixture"
}

func TestRegistryExposesAllFourSurfacesThroughProtocolV6(t *testing.T) {
	t.Parallel()
	p := providercore.NewWithRegistry("test", providercore.Registry{
		Resources:     []func() resource.Resource{func() resource.Resource { return &resourceFixture{} }},
		DataSources:   []func() datasource.DataSource{func() datasource.DataSource { return &dataSourceFixture{} }},
		ListResources: []func() list.ListResource{func() list.ListResource { return &listFixture{} }},
		Actions:       []func() action.Action{func() action.Action { return &actionFixture{} }},
	})
	server := providerserver.NewProtocol6(p)()
	metadata, err := server.GetMetadata(context.Background(), &tfprotov6.GetMetadataRequest{})
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	for _, diagnostic := range metadata.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("metadata diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	if len(metadata.Resources) != 1 || len(metadata.DataSources) != 1 || len(metadata.ListResources) != 1 || len(metadata.Actions) != 1 {
		t.Fatalf("expected one registration for every surface: %#v", metadata)
	}
	if metadata.Resources[0].TypeName != "activedirectory_fixture" || metadata.ListResources[0].TypeName != metadata.Resources[0].TypeName {
		t.Fatalf("list registration must match the managed resource identity: %#v", metadata)
	}
}

func TestRegistryConstructionIsLazyAndIsolated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	constructed := 0
	registry := providercore.Registry{
		Resources: []func() resource.Resource{func() resource.Resource {
			constructed++
			return &resourceFixture{}
		}},
	}
	first := providercore.NewWithRegistry("first", registry)
	second := providercore.NewWithRegistry("second", registry)
	if constructed != 0 {
		t.Fatal("constructing the provider initialized a service")
	}
	// Mutating the original registry or a returned constructor slice must not affect either instance.
	registry.Resources[0] = nil
	first.Resources(ctx)[0] = nil
	if first.Resources(ctx)[0] == nil || second.Resources(ctx)[0] == nil {
		t.Fatal("provider instances share mutable registry slices")
	}
	first.Resources(ctx)[0]()
	if constructed != 1 {
		t.Fatalf("expected only the explicitly selected service to initialize, got %d", constructed)
	}
	if len(first.DataSources(ctx)) != 0 || len(first.ListResources(ctx)) != 0 || len(first.Actions(ctx)) != 0 {
		t.Fatal("resource-only factory initialized unrelated surfaces")
	}
}

func TestRegistrySlicesAreIndependentForEverySurface(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	registry := providercore.Registry{
		Resources:     []func() resource.Resource{func() resource.Resource { return &resourceFixture{} }},
		DataSources:   []func() datasource.DataSource{func() datasource.DataSource { return &dataSourceFixture{} }},
		ListResources: []func() list.ListResource{func() list.ListResource { return &listFixture{} }},
		Actions:       []func() action.Action{func() action.Action { return &actionFixture{} }},
	}
	p := providercore.NewWithRegistry("test", registry)
	assertIndependentSlice(t, "resources", registry.Resources, func() []func() resource.Resource { return p.Resources(ctx) })
	assertIndependentSlice(t, "data sources", registry.DataSources, func() []func() datasource.DataSource { return p.DataSources(ctx) })
	assertIndependentSlice(t, "list resources", registry.ListResources, func() []func() list.ListResource { return p.ListResources(ctx) })
	assertIndependentSlice(t, "actions", registry.Actions, func() []func() action.Action { return p.Actions(ctx) })
}

func assertIndependentSlice[T any](t *testing.T, surface string, original []func() T, registered func() []func() T) {
	t.Helper()
	original[0] = nil
	registered()[0] = nil
	if registered()[0] == nil {
		t.Errorf("%s registration shares a mutable constructor slice", surface)
	}
}
