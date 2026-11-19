// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
)

var (
	version = "dev"
	commit  = "unknown"
)

// main serves the Active Directory provider using Terraform plugin protocol V6.
func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Run with debugger support enabled")
	flag.Parse()

	ctx := context.Background()
	tflog.Info(ctx, "Starting Active Directory provider", map[string]any{
		"version": version,
		"commit":  commit,
	})

	err := providerserver.Serve(ctx, provider.NewActiveDirectoryProvider(version), providerserver.ServeOpts{
		Address:         "registry.terraform.io/deploymenttheory/activedirectory",
		Debug:           debug,
		ProtocolVersion: 6,
	})
	if err != nil {
		log.Printf("Unable to serve Active Directory provider: %v", err)
		os.Exit(1)
	}
}
