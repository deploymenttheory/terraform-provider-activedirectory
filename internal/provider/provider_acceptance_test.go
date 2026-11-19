// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider"
)

// TestAccProviderFoundation exercises Terraform CLI planning and state without directory infrastructure.
func TestAccProviderFoundation(t *testing.T) {
	// This inline configuration has no fixture directories to symlink. An empty source
	// directory lets the Framework test harness run on Windows without symlink privileges.
	t.Chdir(t.TempDir())
	t.Setenv(resource.EnvTfAccProviderNamespace, "deploymenttheory")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"activedirectory": providerserver.NewProtocol6WithError(provider.NewActiveDirectoryProvider("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = {
      source = "deploymenttheory/activedirectory"
    }
  }
}

provider "activedirectory" {}

output "foundation" {
  value = "protocol-v6"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("foundation", knownvalue.StringExact("protocol-v6")),
				},
			},
		},
	})
}
