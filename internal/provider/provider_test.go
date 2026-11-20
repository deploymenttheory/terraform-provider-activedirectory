// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

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
	if schema.ResourceSchemas["activedirectory_organizational_unit"] == nil || schema.DataSourceSchemas["activedirectory_organizational_unit"] == nil ||
		schema.DataSourceSchemas["activedirectory_organizational_units"] == nil || schema.ListResourceSchemas["activedirectory_organizational_unit"] == nil {
		t.Fatal("protocol schema omitted an organizational unit surface")
	}
	if schema.ResourceSchemas["activedirectory_user"] == nil || schema.DataSourceSchemas["activedirectory_user"] == nil ||
		schema.DataSourceSchemas["activedirectory_users"] == nil || schema.ListResourceSchemas["activedirectory_user"] == nil {
		t.Fatal("user lifecycle and discovery schemas were not registered")
	}
	if schema.ResourceSchemas["activedirectory_group"] == nil || schema.DataSourceSchemas["activedirectory_group"] == nil ||
		schema.DataSourceSchemas["activedirectory_groups"] == nil || schema.ListResourceSchemas["activedirectory_group"] == nil {
		t.Fatal("production group resource and data-source registrations did not match their schemas")
	}

	if schema.ResourceSchemas["activedirectory_object"] == nil || schema.ListResourceSchemas["activedirectory_object"] == nil ||
		schema.DataSourceSchemas["activedirectory_object"] == nil || schema.DataSourceSchemas["activedirectory_objects"] == nil {
		t.Fatal("production generic object surfaces were not registered")
	}
	if schema.ResourceSchemas["activedirectory_computer"] == nil || schema.DataSourceSchemas["activedirectory_computer"] == nil ||
		schema.DataSourceSchemas["activedirectory_computers"] == nil || schema.ListResourceSchemas["activedirectory_computer"] == nil {
		t.Fatal("protocol schema omitted a computer surface")
	}

	metadata, err := server.GetMetadata(ctx, &tfprotov6.GetMetadataRequest{})
	if err != nil {
		t.Fatalf("get provider metadata: %v", err)
	}
	assertNoProtocolErrors(t, metadata.Diagnostics)
	if schema.ResourceSchemas["activedirectory_service_account"] == nil || schema.ListResourceSchemas["activedirectory_service_account"] == nil ||
		schema.DataSourceSchemas["activedirectory_service_account"] == nil || schema.DataSourceSchemas["activedirectory_service_accounts"] == nil {
		t.Fatal("production managed service account surfaces were not registered")
	}
	if schema.ResourceSchemas["activedirectory_group_member"] == nil || schema.ListResourceSchemas["activedirectory_group_member"] == nil ||
		schema.DataSourceSchemas["activedirectory_group_member"] == nil || schema.DataSourceSchemas["activedirectory_group_member_relationships"] == nil {
		t.Fatal("production direct membership surfaces were not registered")
	}
	if len(metadata.Resources) < 7 || len(metadata.DataSources) < 14 || len(metadata.ListResources) < 7 {
		t.Fatal("protocol metadata omitted registered directory services")
	}
	if len(metadata.Actions) != 1 || metadata.Actions[0].TypeName != "activedirectory_account_unlock" {
		t.Fatal("protocol metadata omitted the registered account unlock action")
	}

	configType := schema.Provider.ValueType().(tftypes.Object)
	attributes := make(map[string]tftypes.Value, len(configType.AttributeTypes))
	for name, attributeType := range configType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}
	// Explicitly invalid authentication ensures this protocol check never consumes live environment credentials.
	attributes["auth_method"] = tftypes.NewValue(tftypes.String, "unsupported-auth")
	config, err := tfprotov6.NewDynamicValue(configType, tftypes.NewValue(configType, attributes))
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
	if len(configured.Diagnostics) == 0 {
		t.Fatal("unconfigured credentials unexpectedly established a directory client")
	}
}

func assertNoProtocolErrors(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("protocol diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}
