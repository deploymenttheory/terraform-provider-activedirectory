// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/x509"
	"math"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
)

// constructClientConfig validates inputs before client construction or credential file access.
func constructClientConfig(model ActiveDirectoryProviderModel) (config.Config, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	read := func(value types.String, name, environment, fallback string) string {
		if value.IsUnknown() {
			diagnostics.AddAttributeError(path.Root(name), "Unknown provider configuration", "This value must be known before configuring the directory client.")
			return ""
		}
		if !value.IsNull() {
			return value.ValueString()
		}
		if environment != "" {
			if resolved, exists := os.LookupEnv(environment); exists && resolved != "" {
				return resolved
			}
		}
		return fallback
	}
	options := config.Config{
		Endpoint:  config.Endpoint{Host: read(model.Host, "host", "AD_HOST", "")},
		Partition: read(model.Partition, "partition", "AD_PARTITION", ""),
		Timeout:   30 * time.Second,
	}
	backend := read(model.Backend, "backend", "", "portable")
	transport := read(model.Transport, "transport", "AD_TRANSPORT", "ldaps")
	auth := read(model.AuthMethod, "auth_method", "AD_AUTH_METHOD", "")
	protection := read(model.SASLProtection, "sasl_protection", "AD_SASL_PROTECTION", "default")
	username := read(model.Username, "username", "AD_USERNAME", "")
	password := read(model.Password, "password", "AD_PASSWORD", "")
	domain := read(model.Domain, "domain", "", "")
	caPEM := read(model.CACertPEM, "ca_cert_pem", "AD_CA_CERT_PEM", "")
	kerberos := &config.KerberosCredentials{
		Username:   username,
		Realm:      read(model.KerberosRealm, "kerberos_realm", "AD_KERBEROS_REALM", ""),
		SPN:        read(model.KerberosSPN, "kerberos_spn", "AD_KERBEROS_SPN", ""),
		KeytabPath: read(model.KerberosKeytabFile, "kerberos_keytab_file", "AD_KERBEROS_KEYTAB_FILE", ""),
		CachePath:  read(model.KerberosCredentialCacheFile, "kerberos_credential_cache_file", "AD_KERBEROS_CREDENTIAL_CACHE_FILE", ""),
	}
	source := read(model.KerberosCredentialSource, "kerberos_credential_source", "AD_KERBEROS_CREDENTIAL_SOURCE", "")
	options.Backend = choose(&diagnostics, "backend", backend, map[string]config.Backend{"portable": config.BackendPortable, "native": config.BackendNative})
	options.Transport = choose(&diagnostics, "transport", transport, map[string]config.Transport{
		"ldap": config.TransportLDAP, "ldaps": config.TransportLDAPS, "starttls": config.TransportStartTLS,
	})
	options.Authentication = choose(&diagnostics, "auth_method", auth, map[string]config.AuthenticationMethod{
		"simple": config.AuthenticationSimple, "kerberos": config.AuthenticationKerberos, "negotiate": config.AuthenticationNegotiate,
	})
	options.SASLProtection = choose(&diagnostics, "sasl_protection", protection, map[string]config.SASLProtection{
		"default": config.SASLProtectionDefault, "none": config.SASLProtectionNone, "sign": config.SASLProtectionSign, "seal": config.SASLProtectionSeal,
	})
	constructNumericOptions(model, &options, &diagnostics)
	if auth == "kerberos" {
		kerberos.Source = choose(&diagnostics, "kerberos_credential_source", source, map[string]config.KerberosCredentialSource{
			"password": config.KerberosPassword, "keytab": config.KerberosKeytabFile, "credential_cache": config.KerberosCredentialCacheFile,
		})
		if password != "" {
			kerberos.Password = []byte(password)
		}
		options.Kerberos = kerberos
		if domain != "" {
			diagnostics.AddAttributeError(path.Root("domain"), "Incompatible authentication inputs", "Omit domain when using explicit Kerberos realm and service identity.")
		}
	} else {
		if username != "" || password != "" || domain != "" {
			options.Credentials = &config.Credentials{Username: username, Password: []byte(password), Domain: domain}
		}
		if source != "" || kerberos.Realm != "" || kerberos.SPN != "" || kerberos.KeytabPath != "" || kerberos.CachePath != "" {
			diagnostics.AddError("Incompatible authentication inputs", "Kerberos settings require auth_method = kerberos.")
		}
	}
	if caPEM != "" {
		roots, err := x509.SystemCertPool()
		switch {
		case err != nil:
			diagnostics.AddError("Unable to load certificate trust", "The system certificate trust store could not be loaded.")
		case !roots.AppendCertsFromPEM([]byte(caPEM)):
			diagnostics.AddAttributeError(path.Root("ca_cert_pem"), "Invalid certificate trust", "Provide PEM-encoded CA certificates.")
		default:
			options.TLS = &config.TLSOptions{RootCAs: roots}
		}
	}
	if backend == "portable" && auth == "negotiate" {
		diagnostics.AddAttributeError(path.Root("auth_method"), "Unsupported authentication", "Portable execution requires explicit simple or Kerberos authentication.")
	}
	if !diagnostics.HasError() {
		if err := options.Validate(); err != nil {
			diagnostics.AddError("Invalid directory configuration", err.Error())
		}
	}
	return options, diagnostics
}

func constructNumericOptions(model ActiveDirectoryProviderModel, options *config.Config, diagnostics *diag.Diagnostics) {
	if model.Port.IsUnknown() || model.TimeoutSeconds.IsUnknown() {
		diagnostics.AddError("Unknown provider configuration", "Port and timeout_seconds must be known before configuring the directory client.")
	}
	if !model.Port.IsNull() && !model.Port.IsUnknown() {
		port := model.Port.ValueInt64()
		if port < 1 || port > math.MaxUint16 {
			diagnostics.AddAttributeError(path.Root("port"), "Invalid directory port", "Port must be between 1 and 65535.")
		} else {
			options.Endpoint.Port = uint16(port)
		}
	}
	if !model.TimeoutSeconds.IsNull() && !model.TimeoutSeconds.IsUnknown() {
		seconds := model.TimeoutSeconds.ValueInt64()
		if seconds < 1 || seconds > math.MaxInt32 {
			diagnostics.AddAttributeError(path.Root("timeout_seconds"), "Invalid directory timeout", "Timeout must be between 1 and 2147483647 seconds.")
		} else {
			options.Timeout = time.Duration(seconds) * time.Second
		}
	}
}

func choose[T any](diagnostics *diag.Diagnostics, name, value string, choices map[string]T) T {
	selected, ok := choices[value]
	if !ok {
		diagnostics.AddAttributeError(path.Root(name), "Invalid provider option", "Select one of the documented values for this attribute.")
	}
	return selected
}
