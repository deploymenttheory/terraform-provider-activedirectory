// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
)

// Configure validates configuration and establishes one protected SDK session shared by all surfaces.
// Unknown inputs never fall back to environment variables, and credentials are not retained by the provider.
func (p *ActiveDirectoryProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		resp.Diagnostics.AddError("Provider is stopped", "The directory client cannot be configured after provider shutdown.")
		return
	}
	var model ActiveDirectoryProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options, diagnostics := constructClientConfig(model)
	defer clearCredentials(options)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	if p.clients != nil {
		if err := p.clients.Close(); err != nil {
			resp.Diagnostics.AddError("Unable to close previous directory session", "The previous session could not be closed safely. Start a new Terraform provider process.")
			return
		}
		p.clients = nil
	}
	if p.open == nil {
		resp.Diagnostics.AddError("Missing directory client factory", "The provider client factory was not configured.")
		return
	}
	bundle, err := p.open(ctx, options)
	if err != nil || bundle == nil {
		if bundle != nil {
			if closeErr := bundle.Close(); closeErr != nil {
				resp.Diagnostics.AddWarning("Directory cleanup failed", "A failed client factory returned a session that could not be closed safely.")
			}
		}
		// Factory and protocol errors may contain raw peer messages or secrets. Never reflect them into diagnostics.
		resp.Diagnostics.AddError("Unable to configure directory client", "The protected directory session could not be established or qualified. "+
			"Check endpoint connectivity, explicit authentication inputs, certificate trust and the selected transport/security policy.")
		return
	}
	p.clients = bundle
	resp.ResourceData = bundle
	resp.DataSourceData = bundle
	resp.ListResourceData = bundle
	resp.ActionData = bundle
}

// Close retires this provider's session without affecting other provider aliases or instances.
func (p *ActiveDirectoryProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if err := p.clients.Close(); err != nil {
		return fmt.Errorf("close provider clients: %w", err)
	}
	p.clients = nil
	return nil
}

func clearCredentials(options config.Config) {
	if options.Credentials != nil {
		clear(options.Credentials.Password)
	}
	if options.Kerberos != nil {
		clear(options.Kerberos.Password)
	}
}
