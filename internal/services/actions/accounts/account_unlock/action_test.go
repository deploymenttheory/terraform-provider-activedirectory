// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/actions/accounts/account_unlock/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

var errPrivateFixture = errors.New("fixture-password-must-never-be-printed")

//go:embed tests/responses/account.json
var accountResponse []byte

func TestValidationChecksKnownFieldsBesideUnknownReferences(t *testing.T) {
	for _, test := range []struct {
		guid, server types.String
		wantError    bool
	}{
		{guid: types.StringUnknown(), server: types.StringValue("dc.unit.invalid")},
		{guid: types.StringValue(fixtureGUID), server: types.StringUnknown()},
		{guid: types.StringValue("not-a-guid"), server: types.StringUnknown(), wantError: true},
		{guid: types.StringUnknown(), server: types.StringValue("ldap://dc.unit.invalid"), wantError: true},
	} {
		var response action.ValidateConfigResponse
		(&AccountUnlockAction{}).ValidateConfig(t.Context(), action.ValidateConfigRequest{
			Config: actionConfig(t, AccountUnlockActionModel{ObjectGUID: test.guid, Server: test.server}),
		}, &response)
		if response.Diagnostics.HasError() != test.wantError {
			t.Fatalf("known fields were not validated independently: %v", response.Diagnostics)
		}
	}
}

func TestInvocationOutcomesAndRedaction(t *testing.T) {
	for _, test := range []struct {
		name      string
		state     shared.ObjectWriteState
		err       error
		verified  bool
		wantError bool
	}{
		{name: "verified", state: shared.ObjectWriteAccepted, verified: true},
		{name: "accepted-unverifiable", state: shared.ObjectWriteAccepted, err: shared.ErrObjectUnverifiable},
		{name: "rejected", state: shared.ObjectWriteRejected, err: errors.Join(shared.ErrObjectWritePermission, errPrivateFixture), wantError: true},
		{name: "uncertain", state: shared.ObjectWriteUncertain, err: errors.Join(shared.ErrObjectWriteUncertain, errPrivateFixture), wantError: true},
		{name: "accepted-readback-failed", state: shared.ObjectWriteAccepted, err: errors.Join(shared.ErrObjectReadback, errPrivateFixture), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := &featuremocks.Service{
				Unlock: func(_ context.Context, target shared.Identity, options shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
					calls++
					if target.Form() != shared.IdentityFormGUID || target.String() != fixtureGUID || options.ExpectedVersion != nil {
						t.Fatal("unlock changed identity selection or added an assertion control")
					}
					result := fixtureResult(t)
					result.Mutation.State, result.Mutation.Verified = test.state, test.verified
					return result, test.err
				},
			}
			a := &AccountUnlockAction{client: service, server: "dc.unit.invalid", capabilities: shared.DirectoryCapabilities{Type: shared.DirectoryADDS}}
			var response action.InvokeResponse
			a.Invoke(t.Context(), action.InvokeRequest{Config: actionConfig(t, validModel())}, &response)
			if calls != 1 || response.Diagnostics.HasError() != test.wantError {
				t.Fatalf("calls=%d diagnostics=%v", calls, response.Diagnostics)
			}
			for _, diagnostic := range response.Diagnostics {
				if strings.Contains(diagnostic.Detail(), errPrivateFixture.Error()) {
					t.Fatal("peer error disclosed a credential")
				}
			}
		})
	}
}

func TestInvocationRefusalsPreventSDKCalls(t *testing.T) {
	for _, scenario := range []string{"unknown-guid", "null-guid", "invalid-guid", "zero-guid", "unknown-server", "wrong-server", "unsupported", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			model := validModel()
			a := &AccountUnlockAction{server: "dc.unit.invalid", capabilities: shared.DirectoryCapabilities{Type: shared.DirectoryADDS}, client: &featuremocks.Service{
				Unlock: func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
					t.Fatal("refused invocation reached the SDK")
					return accounts.AccountControlMutationResult{}, nil
				},
			}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch scenario {
			case "unknown-guid":
				model.ObjectGUID = types.StringUnknown()
			case "null-guid":
				model.ObjectGUID = types.StringNull()
			case "invalid-guid":
				model.ObjectGUID = types.StringValue("Administrator")
			case "zero-guid":
				model.ObjectGUID = types.StringValue("00000000-0000-0000-0000-000000000000")
			case "unknown-server":
				model.Server = types.StringUnknown()
			case "wrong-server":
				model.Server = types.StringValue("different.unit.invalid")
			case "unsupported":
				a.capabilities.Type = shared.DirectoryADLDS
			case "canceled":
				cancel()
			}
			var response action.InvokeResponse
			a.Invoke(ctx, action.InvokeRequest{Config: actionConfig(t, model)}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("invalid invocation was accepted")
			}
		})
	}
}

func TestInvocationRejectsContradictoryTypedReadback(t *testing.T) {
	for _, scenario := range []string{"nonzero-lockout", "different-guid"} {
		t.Run(scenario, func(t *testing.T) {
			result := fixtureResult(t)
			if scenario == "nonzero-lockout" {
				nonzero := shared.FileTime(1)
				result.Account.LockoutTime = &nonzero
			} else {
				other, _ := shared.ParseGUID("4cdd7c9a-d7bc-4c87-a45d-926c57b6358b")
				// Keep generic mutation readback intact to exercise the typed projection independently.
				result.Account.Raw.Attributes = map[string][][]byte{"objectguid": {other.Bytes()}}
			}
			a := &AccountUnlockAction{server: "dc.unit.invalid", capabilities: shared.DirectoryCapabilities{Type: shared.DirectoryADDS}, client: &featuremocks.Service{
				Unlock: func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
					return result, nil
				},
			}}
			var response action.InvokeResponse
			a.Invoke(t.Context(), action.InvokeRequest{Config: actionConfig(t, validModel())}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("contradictory readback was reported as a completed unlock")
			}
		})
	}
}

func TestTerraformUnitActionTriggerAndNoChange(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/action.tf")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	service := &featuremocks.Service{Unlock: func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
		calls.Add(1)
		return fixtureResult(t), nil
	}}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: string(config), Check: func(*terraform.State) error {
			if calls.Load() != 1 {
				t.Fatal("Terraform action trigger did not submit exactly one unlock")
			}
			return nil
		}},
		{Config: string(config), PlanOnly: true},
	}})
	if calls.Load() != 1 {
		t.Fatal("plan or destroy replayed the action")
	}
}

func TestTerraformUnitRefusalsAndExplicitRecovery(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	t.Setenv("TF_VAR_reader_upn", "reader@unit.invalid")
	t.Setenv("TF_VAR_reader_password", "synthetic-reader-password")
	const connection = "host = \"dc.unit.invalid\"\n auth_method = \"simple\"\n transport = \"ldaps\"\n"
	configuration := func(provider, server string) string {
		return strings.NewReplacer(
			"__PROVIDER__", provider, "__GUID__", `"`+fixtureGUID+`"`, "__SERVER__", `"`+server+`"`,
			`provider "activedirectory" {}`, "provider \"activedirectory\" {\n"+connection+"username = \"fixture@unit.invalid\"\npassword = \"synthetic-password\"\n}",
			"provider \"activedirectory\" {\n  alias", "provider \"activedirectory\" {\n"+connection+"  alias",
		).Replace(labActionConfiguration)
	}
	var calls atomic.Int32
	service := &featuremocks.Service{Unlock: func(context.Context, shared.Identity, shared.ObjectMutationOptions) (accounts.AccountControlMutationResult, error) {
		result := fixtureResult(t)
		if calls.Add(1) == 1 {
			result.Mutation.State, result.Mutation.Verified = shared.ObjectWriteRejected, false
			return result, shared.ErrObjectWritePermission
		}
		return result, nil
	}}
	t.Chdir(t.TempDir())
	admin := configuration("activedirectory", "dc.unit.invalid")
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: configuration("activedirectory", "different.unit.invalid"), ExpectError: regexp.MustCompile("Account unlock server differs")},
		{Config: configuration("activedirectory.reader", "dc.unit.invalid"), ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: admin},
		{Config: admin, PlanOnly: true},
	}})
	if calls.Load() != 2 {
		t.Fatal("a refused target reached the SDK or failed submission was automatically replayed")
	}
}

func validModel() AccountUnlockActionModel {
	return AccountUnlockActionModel{ObjectGUID: types.StringValue(fixtureGUID), Server: types.StringValue("dc.unit.invalid")}
}

func actionConfig(t *testing.T, model AccountUnlockActionModel) tfsdk.Config {
	t.Helper()
	var schemaResponse action.SchemaResponse
	(&AccountUnlockAction{}).Schema(t.Context(), action.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diagnostics := state.Set(t.Context(), model); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return tfsdk.Config{Schema: schemaResponse.Schema, Raw: state.Raw}
}

func fixtureResult(t *testing.T) accounts.AccountControlMutationResult {
	t.Helper()
	var fixture struct {
		GUID              string `json:"guid"`
		DistinguishedName string `json:"distinguishedName"`
		LockoutTime       uint64 `json:"lockoutTime"`
		Enabled           bool   `json:"enabled"`
	}
	if err := json.Unmarshal(accountResponse, &fixture); err != nil {
		t.Fatal(err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		t.Fatal(err)
	}
	zero := shared.FileTime(fixture.LockoutTime)
	object := shared.Object{DistinguishedName: fixture.DistinguishedName, Attributes: map[string][][]byte{"objectguid": {guid.Bytes()}, "lockouttime": {[]byte(zero.LDAPString())}}}
	return accounts.AccountControlMutationResult{
		Account:  &accounts.AccountControlState{Raw: object, LockoutTime: &zero, Enabled: &fixture.Enabled},
		Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, After: &object, Verified: true, ReadbackComplete: true},
	}
}
