// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
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
	// A rejected reconfiguration must not leave the previous authenticated identity usable.
	if p.clients != nil {
		if err := p.clients.Close(); err != nil {
			p.closed = true
			resp.Diagnostics.AddError("Unable to close previous directory session", "The previous session could not be closed safely. Start a new Terraform provider process.")
			return
		}
		p.clients = nil
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
		resp.Diagnostics.AddError("Unable to configure directory client", connectionErrorDetail(err))
		return
	}
	p.clients = bundle
	resp.ResourceData = bundle
	resp.DataSourceData = bundle
	resp.ListResourceData = bundle
	resp.ActionData = bundle
}

func connectionErrorDetail(err error) string {
	var kerberos *shared.KerberosError
	if errors.As(err, &kerberos) && kerberos.Code != 0 {
		return fmt.Sprintf(
			"Kerberos authentication returned protocol code %d. Check the explicit account, realm, service identity, encryption support and clock synchronization.",
			kerberos.Code,
		)
	}
	var ldap *shared.LDAPError
	if errors.As(err, &ldap) {
		return fmt.Sprintf("LDAP returned status %d while establishing or qualifying the directory session. Server diagnostic text is withheld to protect credentials.", ldap.Code)
	}
	switch {
	case errors.Is(err, client.ErrProtection):
		return "The established session did not report the required TLS, Kerberos mutual authentication, channel binding or SASL protection."
	case errors.Is(err, shared.ErrTLSNegotiation):
		return "Verified TLS negotiation failed. Check the certificate chain, endpoint hostname, private CA trust and TLS support."
	case errors.Is(err, shared.ErrKerberosExpired):
		return "The explicit Kerberos credential or context has expired. Supply a current credential source; the provider does not renew it."
	case errors.Is(err, shared.ErrKerberosAuthentication):
		return "Kerberos authentication failed. Check the explicit principal, realm, service identity, credential source and clock synchronization."
	case errors.Is(err, shared.ErrUnsupportedDirectory):
		return "The endpoint did not provide a complete Active Directory Domain Services capability observation."
	case errors.Is(err, config.ErrBackendUnavailable):
		return "The selected SDK backend cannot provide this authentication or protection mechanism on the execution host."
	case errors.Is(err, context.DeadlineExceeded):
		return "The directory connection or qualification read exceeded its timeout. Check endpoint and KDC connectivity."
	case errors.Is(err, context.Canceled):
		return "The directory connection or qualification read was canceled. No authentication fallback was attempted."
	}
	return "The protected directory session could not be established or qualified. " +
		"Check endpoint connectivity, explicit authentication inputs, certificate trust and the selected transport/security policy."
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
