// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/accounts/account_settings/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

// TestTerraformUnitReadOnlySettingsPreservesHiddenValues checks actual GUID/DN reads without registering any managed resource.
func TestTerraformUnitReadOnlySettingsPreservesHiddenValues(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	payload, err := os.ReadFile("tests/responses/account.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		GUID string `json:"guid"`
		DN   string `json:"distinguishedName"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	guid, _ := shared.ParseGUID(fixture.GUID)
	raw := shared.Object{DistinguishedName: fixture.DN, Attributes: map[string][][]byte{"objectguid": {guid.Bytes()}}}
	service := &featuremocks.Service{}
	seen := map[shared.IdentityForm]bool{}
	service.ReadControl = func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
		seen[target.Form()] = true
		locked := false
		return accounts.AccountControlResult{
			Account:     &accounts.AccountControlState{Raw: raw, DistinguishedName: fixture.DN, LockedOut: &locked},
			Observation: shared.IdentityResult{Object: &raw, Complete: true},
		}, nil
	}
	service.ReadExpiration = func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (accounts.AccountExpirationResult, error) {
		if target.Form() != shared.IdentityFormGUID || target.String() != fixtureGUID {
			return accounts.AccountExpirationResult{}, shared.ErrInvalidIdentity
		}
		stamp, never := shared.FileTime(0), true
		return accounts.AccountExpirationResult{
			Expiration:  &accounts.AccountExpiration{Raw: raw, Value: &stamp, NeverExpires: &never},
			Observation: shared.IdentityResult{Object: &raw, Complete: true},
		}, nil
	}
	payload, err = os.ReadFile("tests/terraform/unit/datasource.tf")
	if err != nil {
		t.Fatal(err)
	}
	configuration := string(payload)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: configuration, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".by_guid", "id", fixtureGUID),
			resource.TestCheckResourceAttrPair("data."+DataSourceName+".by_guid", "id", "data."+DataSourceName+".by_dn", "id"),
			resource.TestCheckNoResourceAttr("data."+DataSourceName+".by_guid", "enabled"),
			resource.TestCheckNoResourceAttr("data."+DataSourceName+".by_guid", "password_expired"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".by_guid", "locked_out", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".by_guid", "never_expires", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".by_guid", "account_expires", "0"),
		)},
		{Config: configuration, PlanOnly: true},
	}})
	if !seen[shared.IdentityFormGUID] || !seen[shared.IdentityFormDN] {
		t.Fatal("both public lookup forms were not exercised")
	}
}

func readConfig(t *testing.T, guid, dn any) (tfsdk.Config, tfsdk.State) {
	t.Helper()
	var definition datasource.SchemaResponse
	(&AccountSettingsDataSource{}).Schema(t.Context(), datasource.SchemaRequest{}, &definition)
	objectType := definition.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, kind := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(kind, nil)
	}
	values["object_guid"] = tftypes.NewValue(tftypes.String, guid)
	values["dn"] = tftypes.NewValue(tftypes.String, dn)
	return tfsdk.Config{Schema: definition.Schema, Raw: tftypes.NewValue(objectType, values)}, tfsdk.State{Schema: definition.Schema, Raw: tftypes.NewValue(objectType, nil)}
}

// TestLookupFailuresDoNotReadSDK rejects missing, conflicting, malformed and unresolved inputs before any directory request.
func TestLookupFailuresDoNotReadSDK(t *testing.T) {
	for _, row := range []struct{ guid, dn any }{
		{nil, nil}, {fixtureGUID, "CN=Account,DC=unit,DC=invalid"}, {"00000000-0000-0000-0000-000000000000", nil}, {nil, "not-a-dn"}, {tftypes.UnknownValue, nil},
	} {
		calls := 0
		service := &featuremocks.Service{ReadControl: func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
			calls++
			return accounts.AccountControlResult{}, shared.ErrIncompleteResult
		}}
		config, state := readConfig(t, row.guid, row.dn)
		response := datasource.ReadResponse{State: state}
		(&AccountSettingsDataSource{client: service}).Read(t.Context(), datasource.ReadRequest{Config: config}, &response)
		if !response.Diagnostics.HasError() || calls != 0 || !response.State.Raw.IsNull() {
			t.Fatal("invalid lookup consumed SDK access or emitted partial state")
		}
	}
}

// TestReadDiagnosticsDoNotExposeDirectoryPeerText keeps bind-like private strings out of diagnostics after a real typed read failure.
func TestReadDiagnosticsDoNotExposeDirectoryPeerText(t *testing.T) {
	secret := "synthetic-private-peer-text"
	service := &featuremocks.Service{ReadControl: func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (accounts.AccountControlResult, error) {
		return accounts.AccountControlResult{}, fmt.Errorf("%s: %w", secret, shared.ErrObjectWritePermission)
	}}
	config, state := readConfig(t, fixtureGUID, nil)
	response := datasource.ReadResponse{State: state}
	(&AccountSettingsDataSource{client: service}).Read(t.Context(), datasource.ReadRequest{Config: config}, &response)
	if !response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
		t.Fatal("failed SDK read emitted a usable account snapshot")
	}
	for _, diagnostic := range response.Diagnostics {
		if strings.Contains(diagnostic.Summary()+diagnostic.Detail(), secret) {
			t.Fatal("directory peer text leaked into diagnostics")
		}
	}
}
