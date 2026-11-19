// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
)

func TestProviderFactoryIsolationAndMetadata(t *testing.T) {
	t.Parallel()

	factory := provider.NewActiveDirectoryProvider("0.1.0-test")
	first, second := factory(), factory()
	if first == second {
		t.Fatal("provider factory reused an instance")
	}

	for _, instance := range []frameworkprovider.Provider{first, second} {
		var response frameworkprovider.MetadataResponse
		instance.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &response)
		if response.TypeName != "activedirectory" || response.Version != "0.1.0-test" {
			t.Fatalf("unexpected metadata: %#v", response)
		}
	}
}

func TestProviderProtocolV6(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := providerserver.NewProtocol6(provider.NewActiveDirectoryProvider("test")())()

	schema, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("get provider schema: %v", err)
	}
	assertNoProtocolErrors(t, schema.Diagnostics)
	if schema.Provider == nil || schema.Provider.Block == nil {
		t.Fatal("protocol V6 returned no provider schema")
	}
	if len(schema.ResourceSchemas) != 0 || len(schema.DataSourceSchemas) != 0 {
		t.Fatal("empty foundation unexpectedly registered directory services")
	}

	metadata, err := server.GetMetadata(ctx, &tfprotov6.GetMetadataRequest{})
	if err != nil {
		t.Fatalf("get provider metadata: %v", err)
	}
	assertNoProtocolErrors(t, metadata.Diagnostics)
	if len(metadata.Resources) != 0 || len(metadata.DataSources) != 0 || len(metadata.ListResources) != 0 || len(metadata.Actions) != 0 {
		t.Fatal("empty foundation unexpectedly advertised service metadata")
	}

	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}
	config, err := tfprotov6.NewDynamicValue(configType, tftypes.NewValue(configType, map[string]tftypes.Value{}))
	if err != nil {
		t.Fatalf("encode empty provider configuration: %v", err)
	}
	validated, err := server.ValidateProviderConfig(ctx, &tfprotov6.ValidateProviderConfigRequest{Config: &config})
	if err != nil {
		t.Fatalf("validate provider configuration: %v", err)
	}
	assertNoProtocolErrors(t, validated.Diagnostics)

	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: &config, TerraformVersion: "1.17.2"})
	if err != nil {
		t.Fatalf("configure provider: %v", err)
	}
	assertNoProtocolErrors(t, configured.Diagnostics)
}

func assertNoProtocolErrors(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("protocol diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}
