// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package acceptance_test

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

func TestTerraformUnitUsesSDKBoundaryWithoutDirectoryCredentials(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Chdir(t.TempDir())
	var mu sync.Mutex
	var sessions []*mocks.Session
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		if len(sessions) == 0 {
			t.Error("Terraform never configured the injected SDK factory")
		}
		totalReads := 0
		for _, session := range sessions {
			reads, observations, closed := session.Calls()
			totalReads += reads
			if reads < 1 || observations != 1 || !closed {
				t.Errorf("SDK fixture lifecycle incomplete: reads=%d observations=%d closed=%v", reads, observations, closed)
			}
		}
		if totalReads <= len(sessions) {
			t.Error("Terraform never performed a data-source read beyond client qualification")
		}
	})
	factory := mocks.Factory(func() client.Session {
		name := "DC=unit,DC=invalid"
		session := &mocks.Session{
			RootDSEFunc: func(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
				return shared.RootDSE{DefaultNamingContext: &name, SupportedCapabilities: []string{"1.2.840.113556.1.4.800"}}, nil
			},
			ProtectionFunc: func(context.Context) (activedirectory.SessionProtection, error) {
				return activedirectory.SessionProtection{
					Backend: config.BackendPortable, Authentication: config.AuthenticationSimple, Transport: config.TransportLDAPS,
					Complete: true, TLSVerified: true, TLSVersion: tls.VersionTLS13, TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256,
				}, nil
			},
		}
		mu.Lock()
		sessions = append(sessions, session)
		mu.Unlock()
		return session
	})
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: acceptance.ProviderFactories(
			t,
			providercore.Registry{DataSources: []func() datasource.DataSource{func() datasource.DataSource { return &unitProbe{} }}},
			factory,
		),
		Steps: []resource.TestStep{{Config: `
terraform {
  required_providers { activedirectory = { source = "deploymenttheory/activedirectory" } }
}
provider "activedirectory" {
  host = "unit.invalid"
  auth_method = "simple"
  username = "unit-fixture"
  password = "not-a-real-credential"
}
data "activedirectory_unit_probe" "test" {}
`, Check: resource.TestCheckResourceAttr("data.activedirectory_unit_probe.test", "partition", "DC=unit,DC=invalid")}},
	})
}

type unitProbe struct{ clients *client.Bundle }

func (*unitProbe) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_unit_probe"
}

func (*unitProbe) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{"partition": schema.StringAttribute{Computed: true}}}
}

func (p *unitProbe) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	p.clients, ok = req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected unit fixture data", "Expected injected SDK clients.")
	}
}

func (p *unitProbe) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if p.clients == nil {
		resp.Diagnostics.AddError("Missing unit fixture clients", "Configure did not publish clients.")
		return
	}
	root, err := p.clients.Directory().RootDSE(ctx, shared.RootDSEOptions{})
	if err != nil || root.DefaultNamingContext == nil {
		resp.Diagnostics.AddError("Incomplete unit fixture read", "Expected an injected RootDSE observation.")
		return
	}
	state := struct {
		Partition types.String `tfsdk:"partition"`
	}{types.StringValue(*root.DefaultNamingContext)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
