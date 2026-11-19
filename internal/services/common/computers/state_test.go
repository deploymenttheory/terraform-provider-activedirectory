// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computers

import (
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	sdkcomputers "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func TestProjectionSeparatesEncryptionCompoundAndUnavailableAccountObservations(t *testing.T) {
	guid, _ := shared.ParseGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	name, enabled, compound := "fixture", false, true
	encryption, lastSet, expires := shared.Flags32(0x18), shared.FileTime(0), shared.FileTime(math.MaxInt64)
	computer := sdkcomputers.Computer{
		GUID: &guid, Name: &name, DistinguishedName: "CN=fixture,OU=Computers,DC=unit,DC=invalid",
		ComputerProperties: sdkcomputers.ComputerProperties{KerberosEncryptionTypes: &encryption, CompoundIdentitySupported: &compound, AccountExpires: &expires},
		AccountControl:     accounts.AccountControlState{Enabled: &enabled, PasswordLastSet: &lastSet},
		Raw:                shared.Object{Attributes: map[string][][]byte{"dnshostname": {[]byte("fixture.unit.invalid")}}},
	}
	values, diagnostics := Project(computer)
	if diagnostics.HasError() || values["enabled"].(types.Bool).ValueBool() || !values["compound_identity_supported"].(types.Bool).ValueBool() ||
		!values["change_password_at_logon"].(types.Bool).ValueBool() || !values["password_not_required"].IsNull() ||
		values["account_expires"].(types.Int64).ValueInt64() != math.MaxInt64 || values["dns_host_name"].(types.String).ValueString() != "fixture.unit.invalid" {
		t.Fatal("projection fabricated hidden values or conflated directory policy and host metadata")
	}
	selected := values["kerberos_encryption_types"].(types.Set)
	if selected.IsNull() || len(selected.Elements()) != 2 {
		t.Fatal("named encryption selection included compound or unrelated flags")
	}
	for _, forbidden := range []string{"password_wo", "password_wo_version", "user_account_control", "supported_encryption_types", "service_principal_names", "domain_joined"} {
		if _, found := AttributeTypes()[forbidden]; found {
			t.Fatal("read-only projection exposed secret metadata, a raw mask or unrelated ownership")
		}
	}
	encryption = 0
	values, diagnostics = Project(computer)
	if diagnostics.HasError() || values["kerberos_encryption_types"].IsNull() || len(values["kerberos_encryption_types"].(types.Set).Elements()) != 0 {
		t.Fatal("explicit zero encryption selection became unavailable")
	}
	computer.KerberosEncryptionTypes, computer.AccountControl.PasswordLastSet = nil, nil
	values, diagnostics = Project(computer)
	if diagnostics.HasError() || !values["kerberos_encryption_types"].IsNull() || !values["change_password_at_logon"].IsNull() {
		t.Fatal("hidden encryption or pwdLastSet was invented")
	}
}

func TestProjectionRejectsUnusableIdentityLocationAndExpiration(t *testing.T) {
	guid, _ := shared.ParseGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	name := "fixture"
	for _, computer := range []sdkcomputers.Computer{
		{},
		{GUID: &guid, Name: &name, DistinguishedName: "malformed"},
	} {
		if values, diagnostics := Project(computer); !diagnostics.HasError() || values != nil {
			t.Fatal("unusable identity emitted an import snapshot")
		}
	}
	expires := shared.FileTime(math.MaxUint64)
	values, diagnostics := Project(sdkcomputers.Computer{
		GUID: &guid, Name: &name, DistinguishedName: "CN=fixture,DC=unit,DC=invalid",
		ComputerProperties: sdkcomputers.ComputerProperties{AccountExpires: &expires},
	})
	if !diagnostics.HasError() || !values["account_expires"].IsNull() {
		t.Fatal("unsupported expiration overflowed into an observed negative value")
	}
}
