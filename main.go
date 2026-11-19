// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package main

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name terraform-provider-activedirectory --tf-version 1.17.2

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
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

	if err := serve(debug); err != nil {
		log.Printf("Unable to serve Active Directory provider: %v", err)
		os.Exit(1)
	}
}

func serve(debug bool) error {
	instance := provider.NewActiveDirectoryProvider(version)().(*provider.ActiveDirectoryProvider)
	defer func() {
		if err := instance.Close(); err != nil {
			log.Print("Directory session cleanup failed during provider shutdown")
		}
	}()
	var options []tf6server.ServeOpt
	if debug {
		options = append(options, tf6server.WithManagedDebug())
	}
	err := tf6server.Serve("registry.terraform.io/deploymenttheory/activedirectory", func() tfprotov6.ProviderServer {
		return providercore.NewProtocol6Server(instance)
	}, options...)
	if err != nil {
		return fmt.Errorf("serve Active Directory protocol V6: %w", err)
	}
	return nil
}
