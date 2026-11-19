// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActiveDirectoryProviderModel describes provider inputs, including caller-owned credential values.
type ActiveDirectoryProviderModel struct {
	Host                        types.String `tfsdk:"host"`
	Port                        types.Int64  `tfsdk:"port"`
	Backend                     types.String `tfsdk:"backend"`
	Partition                   types.String `tfsdk:"partition"`
	AuthMethod                  types.String `tfsdk:"auth_method"`
	Transport                   types.String `tfsdk:"transport"`
	Username                    types.String `tfsdk:"username"`
	Password                    types.String `tfsdk:"password"`
	Domain                      types.String `tfsdk:"domain"`
	CACertPEM                   types.String `tfsdk:"ca_cert_pem"`
	TimeoutSeconds              types.Int64  `tfsdk:"timeout_seconds"`
	SASLProtection              types.String `tfsdk:"sasl_protection"`
	KerberosRealm               types.String `tfsdk:"kerberos_realm"`
	KerberosSPN                 types.String `tfsdk:"kerberos_spn"`
	KerberosCredentialSource    types.String `tfsdk:"kerberos_credential_source"`
	KerberosKeytabFile          types.String `tfsdk:"kerberos_keytab_file"`
	KerberosCredentialCacheFile types.String `tfsdk:"kerberos_credential_cache_file"`
}

// Format prevents accidental reflection of provider inputs into formatted logs or diagnostics.
func (ActiveDirectoryProviderModel) Format(state fmt.State, _ rune) {
	_, _ = state.Write([]byte("ActiveDirectoryProviderModel{redacted}"))
}

// MarshalJSON prevents serialization of provider credentials.
func (ActiveDirectoryProviderModel) MarshalJSON() ([]byte, error) {
	return []byte(`{"redacted":true}`), nil
}

// LogValue prevents structured log handlers from reflecting credential fields.
func (ActiveDirectoryProviderModel) LogValue() slog.Value {
	return slog.StringValue("ActiveDirectoryProviderModel{redacted}")
}
