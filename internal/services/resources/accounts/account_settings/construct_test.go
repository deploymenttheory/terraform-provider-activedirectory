// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

// TestConstructionOwnsOnlyConfiguredSettings keeps omission distinct from explicit false and computed status out of every SDK patch.
func TestConstructionOwnsOnlyConfiguredSettings(t *testing.T) {
	t.Parallel()
	model := AccountSettingsResourceModel{
		AccountGUID: types.StringValue(fixtureGUID), Enabled: types.BoolValue(false), AccountNotDelegated: types.BoolValue(true),
		LockedOut: types.BoolValue(true), PasswordExpired: types.BoolValue(true), PasswordLastSet: types.Int64Value(0), AccountExpires: types.Int64Value(0),
	}
	intent, diagnostics := constructSettings(&model)
	if diagnostics.HasError() || intent.GUID != fixtureGUID || intent.Patch.Enabled == nil || *intent.Patch.Enabled ||
		intent.Patch.AccountNotDelegated == nil || !*intent.Patch.AccountNotDelegated || len(intent.Flags) != 2 || intent.Patch.PasswordNotRequired != nil ||
		intent.Patch.ChangePasswordAtLogon != nil || intent.ClearExpiration || intent.ExpiresAt != nil {
		t.Fatal("unconfigured or computed fields entered an account mutation")
	}
}

// TestConstructionDistinguishesFiniteClearAndOmittedExpiration prevents omission or a boolean false from choosing a new expiration.
func TestConstructionDistinguishesFiniteClearAndOmittedExpiration(t *testing.T) {
	t.Parallel()
	model := AccountSettingsResourceModel{AccountGUID: types.StringValue(fixtureGUID), ExpiresAt: types.StringValue("2031-04-05T11:34:56.1234567Z")}
	intent, diagnostics := constructSettings(&model)
	if diagnostics.HasError() || intent.ExpiresAt == nil || intent.ClearExpiration {
		t.Fatal("finite expiration was not retained exactly")
	}
	model.ExpiresAt, model.NeverExpires = types.StringNull(), types.BoolValue(true)
	intent, diagnostics = constructSettings(&model)
	if diagnostics.HasError() || !intent.ClearExpiration || intent.ExpiresAt != nil {
		t.Fatal("explicit never-expire intent did not choose the typed clear operation")
	}
	model.NeverExpires = types.BoolNull()
	intent, diagnostics = constructSettings(&model)
	if diagnostics.HasError() || intent.ClearExpiration || intent.ExpiresAt != nil {
		t.Fatal("omission became an expiration reset")
	}
}

// TestConstructionRefusesUnresolvedAndConflictingPhases validates later expiry input before an earlier flag change can be submitted.
func TestConstructionRefusesUnresolvedAndConflictingPhases(t *testing.T) {
	t.Parallel()
	for _, model := range []AccountSettingsResourceModel{
		{AccountGUID: types.StringUnknown()},
		{AccountGUID: types.StringValue(fixtureGUID), Enabled: types.BoolUnknown()},
		{AccountGUID: types.StringValue(fixtureGUID), Enabled: types.BoolValue(true), ExpiresAt: types.StringUnknown()},
		{AccountGUID: types.StringValue(fixtureGUID), Enabled: types.BoolValue(true), ExpiresAt: types.StringValue("2031-04-05T12:34:56+01:00")},
		{AccountGUID: types.StringValue(fixtureGUID), ExpiresAt: types.StringValue("2031-04-05T11:34:56.123456789Z")},
		{AccountGUID: types.StringValue(fixtureGUID), NeverExpires: types.BoolValue(false)},
		{AccountGUID: types.StringValue(fixtureGUID), ExpiresAt: types.StringValue("2031-04-05T11:34:56Z"), NeverExpires: types.BoolValue(true)},
		{AccountGUID: types.StringValue(fixtureGUID), ChangePasswordAtLogon: types.BoolValue(true), PasswordNeverExpires: types.BoolValue(true)},
	} {
		if _, diagnostics := constructSettings(&model); !diagnostics.HasError() {
			t.Fatal("invalid account settings passed mutation preflight")
		}
	}
}
