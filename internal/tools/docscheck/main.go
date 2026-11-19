// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Command docscheck verifies documentation coverage for the provider's registered schemas.
package main

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	docs "github.com/deploymenttheory/terraform-provider-activedirectory/internal/documentation"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
)

func main() {
	server := providerserver.NewProtocol6(provider.NewActiveDirectoryProvider("docs")())()
	schema, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		log.Fatal("cannot retrieve provider documentation schema")
	}
	for _, diagnostic := range schema.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			log.Fatal("provider schema validation failed")
		}
	}
	if err := docs.CheckInputs(os.DirFS("."), schema); err != nil {
		log.Fatal(err)
	}
}
