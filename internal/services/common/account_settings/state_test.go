// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"math"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func observation(t *testing.T, id string) shared.Object {
	t.Helper()
	guid, err := shared.ParseGUID(id)
	if err != nil {
		t.Fatal(err)
	}
	return shared.Object{DistinguishedName: "CN=Fixture,OU=Owned,DC=unit,DC=invalid", Attributes: map[string][][]byte{"objectguid": {guid.Bytes()}}}
}

// TestProjectionPreservesVisibilityAndExpirationSentinels refuses to turn missing account observations into reset instructions.
func TestProjectionPreservesVisibilityAndExpirationSentinels(t *testing.T) {
	t.Parallel()
	raw := observation(t, "dc93656c-f745-4926-9bbb-31e7bbecb68b")
	enabled, expired := false, true
	zero, never := shared.FileTime(0), shared.FileTime(math.MaxInt64)
	control := accounts.AccountControlState{Raw: raw, DistinguishedName: raw.DistinguishedName, Enabled: &enabled, PasswordExpired: &expired, PasswordLastSet: &zero}
	expiration := accounts.AccountExpiration{Raw: raw, Value: &never}
	values, diagnostics := Project(control, expiration)
	if diagnostics.HasError() || values["enabled"].IsNull() || values["enabled"].(types.Bool).ValueBool() ||
		!values["password_expired"].(types.Bool).ValueBool() || !values["locked_out"].IsNull() ||
		!values["password_not_required"].IsNull() || !values["change_password_at_logon"].(types.Bool).ValueBool() ||
		values["account_expires"].(types.Int64).ValueInt64() != math.MaxInt64 {
		t.Fatal("missing observations, stored flags, computed flags or expiration sentinels were conflated")
	}
	expiration.Value = &zero
	values, diagnostics = Project(control, expiration)
	if diagnostics.HasError() || values["account_expires"].IsNull() || values["account_expires"].(types.Int64).ValueInt64() != 0 {
		t.Fatal("the zero never-expire representation was lost")
	}
	control.PasswordLastSet, expiration.Value = nil, nil
	values, diagnostics = Project(control, expiration)
	if diagnostics.HasError() || !values["account_expires"].IsNull() || !values["change_password_at_logon"].IsNull() {
		t.Fatal("hidden timestamps became clearing instructions")
	}
	for _, forbidden := range []string{"password_wo", "unicode_pwd", "user_account_control", "computed_account_control", "lockout_time"} {
		if _, exists := AttributeTypes()[forbidden]; exists {
			t.Fatal("the schema exposed a secret or writable raw account state")
		}
	}
}

// TestProjectionRefusesCrossIdentityAndInvalidTimestamps prevents two independently resolved accounts from forming one settings snapshot.
func TestProjectionRefusesCrossIdentityAndInvalidTimestamps(t *testing.T) {
	t.Parallel()
	raw := observation(t, "dc93656c-f745-4926-9bbb-31e7bbecb68b")
	other := observation(t, "a3e9767a-fb25-4d2c-849a-14d672c61005")
	control := accounts.AccountControlState{Raw: raw, DistinguishedName: raw.DistinguishedName}
	overflow := shared.FileTime(math.MaxUint64)
	for _, expiration := range []accounts.AccountExpiration{{Raw: other}, {}, {Raw: raw, Value: &overflow}} {
		if values, diagnostics := Project(control, expiration); values != nil || !diagnostics.HasError() {
			t.Fatal("unverified identity or an invalid timestamp emitted state")
		}
	}
	control.PasswordLastSet = &overflow
	if values, diagnostics := Project(control, accounts.AccountExpiration{Raw: raw}); values != nil || !diagnostics.HasError() {
		t.Fatal("an invalid pwdLastSet emitted state")
	}
}

// TestProjectionUsesExplicitUTCExpiry keeps clearing intent separate from omission and preserves finite timestamp precision.
func TestProjectionUsesExplicitUTCExpiry(t *testing.T) {
	t.Parallel()
	raw := observation(t, "dc93656c-f745-4926-9bbb-31e7bbecb68b")
	control := accounts.AccountControlState{Raw: raw, DistinguishedName: raw.DistinguishedName}
	at := time.Date(2031, 4, 5, 12, 34, 56, 123456700, time.FixedZone("fixture", 3600))
	stamp, err := shared.FileTimeFromTime(at)
	if err != nil {
		t.Fatal(err)
	}
	never := false
	expiration := accounts.AccountExpiration{Raw: raw, Value: &stamp, NeverExpires: &never, ExpiresAt: &at}
	values, diagnostics := Project(control, expiration)
	if diagnostics.HasError() || values["never_expires"].IsNull() || values["never_expires"].(types.Bool).ValueBool() ||
		values["expires_at"].(types.String).ValueString() != "2031-04-05T11:34:56.1234567Z" {
		t.Fatal("finite expiration lost UTC or directory precision")
	}
	never = true
	expiration.ExpiresAt = nil
	for _, value := range []shared.FileTime{0, math.MaxInt64} {
		expiration.Value = &value
		values, diagnostics = Project(control, expiration)
		if diagnostics.HasError() || !values["expires_at"].IsNull() || !values["never_expires"].(types.Bool).ValueBool() {
			t.Fatal("never-expire observations became a finite expiration")
		}
	}
}
