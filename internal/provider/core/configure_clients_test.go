// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

type configureSession struct{ closed int }

func (*configureSession) RootDSE(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
	return shared.RootDSE{SupportedCapabilities: []string{"1.2.840.113556.1.4.800"}}, nil
}

func (*configureSession) Protection(context.Context) (activedirectory.SessionProtection, error) {
	return activedirectory.SessionProtection{
		Backend: config.BackendPortable, Authentication: config.AuthenticationSimple, Transport: config.TransportLDAPS,
		TLSVerified: true, TLSVersion: tls.VersionTLS13, TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256, Complete: true,
	}, nil
}
func (s *configureSession) Close() error { s.closed++; return nil }

func validProviderModel() ActiveDirectoryProviderModel {
	return ActiveDirectoryProviderModel{
		Host: types.StringValue("dc.example.test"), AuthMethod: types.StringValue("simple"),
		Username: types.StringValue("test@example.test"), Password: types.StringValue("secret-unit-value"),
	}
}

func isolateEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"AD_HOST", "AD_PARTITION", "AD_TRANSPORT", "AD_AUTH_METHOD", "AD_SASL_PROTECTION", "AD_USERNAME", "AD_PASSWORD", "AD_CA_CERT_PEM",
		"AD_KERBEROS_REALM", "AD_KERBEROS_SPN", "AD_KERBEROS_CREDENTIAL_SOURCE", "AD_KERBEROS_KEYTAB_FILE", "AD_KERBEROS_CREDENTIAL_CACHE_FILE",
	} {
		t.Setenv(name, "")
	}
}

func configureRequest(t *testing.T, p *ActiveDirectoryProvider, model ActiveDirectoryProviderModel) provider.ConfigureRequest {
	t.Helper()
	ctx := context.Background()
	var schema provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schema)
	var value types.Object
	diagnostics := tfsdk.ValueFrom(ctx, model, schema.Schema.Type(), &value)
	if diagnostics.HasError() {
		t.Fatalf("encode provider model: %v", diagnostics)
	}
	raw, err := value.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return provider.ConfigureRequest{Config: tfsdk.Config{Raw: raw, Schema: schema.Schema}, TerraformVersion: "1.17.2"}
}

func TestConfigureSharesOneBundleAndIsolatesAliases(t *testing.T) {
	isolateEnvironment(t)
	var capturedPassword []byte
	var sessions []*configureSession
	factory := func(ctx context.Context, options config.Config) (*client.Bundle, error) {
		capturedPassword = options.Credentials.Password
		session := &configureSession{}
		sessions = append(sessions, session)
		return client.OpenWithDial(ctx, options, func(context.Context, config.Config) (client.Session, error) { return session, nil })
	}
	first := NewWithRegistry("test", Registry{}, WithClientFactory(factory))
	second := NewWithRegistry("test", Registry{}, WithClientFactory(factory))
	var response, other provider.ConfigureResponse
	first.Configure(context.Background(), configureRequest(t, first, validProviderModel()), &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if response.ResourceData == nil || response.ResourceData != response.DataSourceData || response.ResourceData != response.ListResourceData ||
		response.ResourceData != response.ActionData {
		t.Fatal("provider did not publish one bundle to every surface")
	}
	if strings.Trim(string(capturedPassword), "\x00") != "" {
		t.Fatal("caller-owned password buffer was retained after configuration")
	}
	second.Configure(context.Background(), configureRequest(t, second, validProviderModel()), &other)
	if other.Diagnostics.HasError() || response.ResourceData == other.ResourceData {
		t.Fatal("provider aliases shared a session")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if sessions[0].closed != 1 || sessions[1].closed != 0 {
		t.Fatal("closing one alias affected another")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidReconfigurationRetiresPreviousCredentials(t *testing.T) {
	isolateEnvironment(t)
	session := &configureSession{}
	opens := 0
	p := NewWithRegistry("test", Registry{}, WithClientFactory(func(ctx context.Context, options config.Config) (*client.Bundle, error) {
		opens++
		return client.OpenWithDial(ctx, options, func(context.Context, config.Config) (client.Session, error) { return session, nil })
	}))
	var first provider.ConfigureResponse
	p.Configure(context.Background(), configureRequest(t, p, validProviderModel()), &first)
	if first.Diagnostics.HasError() {
		t.Fatal(first.Diagnostics)
	}
	invalid := validProviderModel()
	invalid.Password = types.StringUnknown()
	var second provider.ConfigureResponse
	p.Configure(context.Background(), configureRequest(t, p, invalid), &second)
	if !second.Diagnostics.HasError() || second.ResourceData != nil || opens != 1 {
		t.Fatal("invalid reconfiguration reached a new client or published credentials")
	}
	if session.closed != 1 {
		t.Fatal("previous authenticated identity remained live after invalid reconfiguration")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigureRejectsInvalidInputsWithoutDial(t *testing.T) {
	isolateEnvironment(t)
	cases := map[string]func(*ActiveDirectoryProviderModel){
		"unknown host":                   func(m *ActiveDirectoryProviderModel) { m.Host = types.StringUnknown() },
		"unknown password":               func(m *ActiveDirectoryProviderModel) { m.Password = types.StringUnknown() },
		"missing authentication":         func(m *ActiveDirectoryProviderModel) { m.AuthMethod = types.StringNull() },
		"unprotected simple bind":        func(m *ActiveDirectoryProviderModel) { m.Transport = types.StringValue("ldap") },
		"unsupported portable negotiate": func(m *ActiveDirectoryProviderModel) { m.AuthMethod = types.StringValue("negotiate") },
		"invalid port":                   func(m *ActiveDirectoryProviderModel) { m.Port = types.Int64Value(65536) },
		"timeout overflow":               func(m *ActiveDirectoryProviderModel) { m.TimeoutSeconds = types.Int64Value(2147483648) },
		"bad certificate":                func(m *ActiveDirectoryProviderModel) { m.CACertPEM = types.StringValue("not a certificate") },
		"conflicting credentials":        func(m *ActiveDirectoryProviderModel) { m.KerberosCredentialSource = types.StringValue("keytab") },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			// Unknown explicit configuration must not consume an environment fallback.
			t.Setenv("AD_HOST", "dc.example.test")
			t.Setenv("AD_PASSWORD", "fallback-secret")
			t.Setenv("AD_AUTH_METHOD", "")
			called := false
			p := NewWithRegistry("test", Registry{}, WithClientFactory(func(context.Context, config.Config) (*client.Bundle, error) {
				called = true
				return nil, errors.New("unexpected dial")
			}))
			model := validProviderModel()
			change(&model)
			var response provider.ConfigureResponse
			p.Configure(context.Background(), configureRequest(t, p, model), &response)
			if !response.Diagnostics.HasError() || called || response.ResourceData != nil {
				t.Fatal("invalid/unknown inputs reached the client factory")
			}
		})
	}
}

func TestConfigureDiagnosticsAndFormattingRedactSecrets(t *testing.T) {
	isolateEnvironment(t)
	model := validProviderModel()
	p := NewWithRegistry("test", Registry{}, WithClientFactory(func(context.Context, config.Config) (*client.Bundle, error) {
		return nil, errors.New("peer echoed secret-unit-value")
	}))
	var response provider.ConfigureResponse
	p.Configure(context.Background(), configureRequest(t, p, model), &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected factory failure")
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{fmt.Sprintf("%+v", model), fmt.Sprintf("%#v", model), string(encoded), fmt.Sprint(response.Diagnostics)} {
		if strings.Contains(output, "secret-unit-value") {
			t.Fatal("provider inputs leaked into diagnostics or formatting")
		}
	}
}

func TestConstructKerberosCredentialSources(t *testing.T) {
	isolateEnvironment(t)
	for _, source := range []string{"password", "keytab", "credential_cache"} {
		t.Run(source, func(t *testing.T) {
			model := validProviderModel()
			model.AuthMethod = types.StringValue("kerberos")
			model.Username = types.StringValue("test")
			model.KerberosRealm = types.StringValue("EXAMPLE.TEST")
			model.KerberosSPN = types.StringValue("ldap/dc.example.test")
			model.KerberosCredentialSource = types.StringValue(source)
			if source != "password" {
				model.Password = types.StringValue("")
			}
			if source == "keytab" {
				model.KerberosKeytabFile = types.StringValue("/not-read-during-validation/client.keytab")
			}
			if source == "credential_cache" {
				model.KerberosCredentialCacheFile = types.StringValue("/not-read-during-validation/client.ccache")
			}
			options, diagnostics := constructClientConfig(model)
			defer clearCredentials(options)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if options.Backend != config.BackendPortable || options.Kerberos == nil {
				t.Fatal("explicit portable Kerberos configuration was not constructed")
			}
		})
	}
}

func TestConnectionErrorsExposeSafeCauseWithoutPeerDiagnostics(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err  error
		want string
	}{
		{client.ErrProtection, "required TLS"},
		{shared.ErrTLSNegotiation, "Verified TLS negotiation failed"},
		{shared.ErrKerberosExpired, "has expired"},
		{shared.ErrKerberosAuthentication, "Kerberos authentication failed"},
		{&shared.KerberosError{Code: 14}, "Kerberos authentication returned protocol code 14"},
		{shared.ErrUnsupportedDirectory, "capability observation"},
		{config.ErrBackendUnavailable, "execution host"},
		{context.DeadlineExceeded, "exceeded its timeout"},
		{context.Canceled, "was canceled"},
		{&shared.LDAPError{Code: 49, Diagnostic: "secret-unit-value", Operation: "secret-unit-value", MatchedDN: "secret-unit-value"}, "LDAP returned status 49"},
	}
	for _, test := range cases {
		detail := connectionErrorDetail(errors.Join(test.err, errors.New("peer echoed secret-unit-value")))
		if !strings.Contains(detail, test.want) || strings.Contains(detail, "secret-unit-value") {
			t.Fatalf("safe error classification failed: %s", detail)
		}
	}
}
