// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

// Schema defines explicit directory connection and authentication inputs.
func (p *ActiveDirectoryProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Active Directory Domain Services through one explicitly protected SDK session. " +
			"Credentials are provider configuration and are never stored in resource state. No authentication or transport fallback is performed.",
		Attributes: map[string]schema.Attribute{
			"host":    stringAttribute("LDAP server FQDN or IP address, without a URL scheme or port. May be set with AD_HOST.", false),
			"port":    schema.Int64Attribute{Optional: true, MarkdownDescription: "LDAP port. Defaults to 636 for LDAPS and 389 for LDAP/StartTLS."},
			"backend": stringAttribute("SDK backend: portable (default) or native. Native is available only on supported Windows hosts.", false),
			"partition": stringAttribute(
				"Explicit naming context DN. Otherwise the SDK uses the server's default naming context. May be set with AD_PARTITION.",
				false,
			),
			"auth_method": stringAttribute(
				"Explicit authentication: kerberos or simple; negotiate is available only with backend = native. May be set with AD_AUTH_METHOD.",
				false,
			),
			"transport": stringAttribute(
				"Transport: ldaps (default), starttls or ldap. LDAP requires Kerberos signing/sealing. May be set with AD_TRANSPORT.",
				false,
			),
			"username": stringAttribute("Explicit bind name or Kerberos principal name without the realm suffix. May be set with AD_USERNAME.", true),
			"password": stringAttribute(
				"Password for simple bind, explicit native Negotiate, or Kerberos password authentication. May be set with AD_PASSWORD.",
				true,
			),
			"domain": stringAttribute("Domain for explicit native Negotiate credentials. Omit for simple bind and Kerberos.", false),
			"ca_cert_pem": stringAttribute(
				"PEM CA certificates added to system trust for portable TLS. Certificate and hostname verification are mandatory. May be set with AD_CA_CERT_PEM.",
				false,
			),
			"timeout_seconds": schema.Int64Attribute{Optional: true, MarkdownDescription: "Operation timeout in seconds, from 1 to 2147483647. Defaults to 30."},
			"sasl_protection": stringAttribute(
				"Kerberos LDAP traffic protection: sign or seal (default). Over TLS use none or omit. May be set with AD_SASL_PROTECTION.",
				false,
			),
			"kerberos_realm": stringAttribute("Explicit Kerberos DNS realm. May be set with AD_KERBEROS_REALM.", false),
			"kerberos_spn":   stringAttribute("Explicit service identity ldap/<endpoint-FQDN>. May be set with AD_KERBEROS_SPN.", false),
			"kerberos_credential_source": stringAttribute(
				"Explicit Kerberos credential source: password, keytab or credential_cache. May be set with AD_KERBEROS_CREDENTIAL_SOURCE.",
				false,
			),
			"kerberos_keytab_file": stringAttribute("Caller-owned keytab path for the keytab credential source. May be set with AD_KERBEROS_KEYTAB_FILE.", true),
			"kerberos_credential_cache_file": stringAttribute(
				"Caller-owned credential cache path. No ambient cache lookup. May be set with AD_KERBEROS_CREDENTIAL_CACHE_FILE.",
				true,
			),
		},
	}
}

func stringAttribute(description string, sensitive bool) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Sensitive: sensitive, MarkdownDescription: description}
}
