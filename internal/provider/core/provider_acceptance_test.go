// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

// TestAccProviderLabConnection exercises the real provider Configure and SDK session through Terraform CLI.
// The connection probe is a test fixture, never registered as a production data source.
func TestAccProviderLabConnection(t *testing.T) {
	if os.Getenv("TF_ACC_AD_LAB") != "1" {
		t.Skip("set TF_ACC=1 and TF_ACC_AD_LAB=1 to run against the explicitly configured lab")
	}
	target := acceptance.PreCheck(t)
	t.Chdir(t.TempDir())
	t.Setenv(resource.EnvTfAccProviderNamespace, "deploymenttheory")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"activedirectory": func() (tfprotov6.ProviderServer, error) {
				p := providercore.NewWithRegistry("test", providercore.Registry{
					DataSources: []func() datasource.DataSource{func() datasource.DataSource { return &connectionProbe{} }},
				})
				t.Cleanup(func() {
					if err := p.Close(); err != nil {
						t.Error("close acceptance directory client")
					}
				})
				return providercore.NewProtocol6Server(p), nil
			},
		},
		Steps: []resource.TestStep{{
			Config: `
terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}
provider "activedirectory" {}
data "activedirectory_connection_probe" "lab" {}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.activedirectory_connection_probe.lab", "server_host", strings.ToLower(target.Host)),
				resource.TestCheckResourceAttr("data.activedirectory_connection_probe.lab", "partition", target.Partition),
				resource.TestCheckResourceAttr("data.activedirectory_connection_probe.lab", "protected", "true"),
			),
		}},
	})
}

type connectionProbe struct{ clients *client.Bundle }

func (*connectionProbe) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection_probe"
}

func (*connectionProbe) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{Attributes: map[string]datasourceschema.Attribute{
		"server_host": datasourceschema.StringAttribute{Computed: true},
		"partition":   datasourceschema.StringAttribute{Computed: true},
		"protected":   datasourceschema.BoolAttribute{Computed: true},
	}}
}

func (p *connectionProbe) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	p.clients, ok = req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected fixture provider data", "Expected the shared directory client bundle.")
	}
}

func (p *connectionProbe) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if p.clients == nil {
		resp.Diagnostics.AddError("Missing fixture client", "Provider Configure did not publish a client.")
		return
	}
	root, err := p.clients.Directory().RootDSE(ctx, shared.RootDSEOptions{})
	if err != nil || root.DNSHostName == nil || root.DefaultNamingContext == nil {
		resp.Diagnostics.AddError("Incomplete fixture observation", "The authenticated RootDSE read did not return the expected lab identity.")
		return
	}
	observed := p.clients.Protection()
	protected := observed.Complete && (observed.TLSVerified || observed.SASLProtection == config.SASLProtectionSign || observed.SASLProtection == config.SASLProtectionSeal)
	state := struct {
		ServerHost types.String `tfsdk:"server_host"`
		Partition  types.String `tfsdk:"partition"`
		Protected  types.Bool   `tfsdk:"protected"`
	}{types.StringValue(strings.ToLower(*root.DNSHostName)), types.StringValue(*root.DefaultNamingContext), types.BoolValue(protected)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
