// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_accounts

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// accountFixture supplies a usable directory identity without inventing optional attribute visibility.
func accountFixture() sdkaccounts.ServiceAccount {
	guid, _ := shared.ParseGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	name := "managed"
	return sdkaccounts.ServiceAccount{
		Kind: sdkaccounts.ServiceAccountGroupManaged, GUID: &guid, Name: &name,
		DistinguishedName: "CN=managed,OU=Service Accounts,DC=unit,DC=invalid",
	}
}

// TestProjectionPreservesReaderVisibility distinguishes hidden ACLs from a returned empty reader policy.
func TestProjectionPreservesReaderVisibility(t *testing.T) {
	account := accountFixture()
	values, diagnostics := Project(account)
	if diagnostics.HasError() || !values["password_reader_sids"].IsNull() || !values["service_principal_names"].IsNull() || !values["enabled"].IsNull() {
		t.Fatal("unavailable metadata became an empty collection or a false flag")
	}
	account.PasswordReaderDescriptorReturned = true
	account.ServicePrincipalNames = shared.AttributeValues{State: shared.AttributePresent}
	values, diagnostics = Project(account)
	if diagnostics.HasError() || values["password_reader_sids"].IsNull() || len(values["password_reader_sids"].(types.Set).Elements()) != 0 ||
		values["service_principal_names"].IsNull() {
		t.Fatal("positively returned empty policy was treated as hidden")
	}
	account.ServicePrincipalNames = shared.AttributeValues{State: shared.AttributePartial, Values: []shared.RawValue{shared.NewRawValue([]byte("HOST/partial.unit.invalid"))}}
	values, diagnostics = Project(account)
	if diagnostics.HasError() || !values["service_principal_names"].IsNull() {
		t.Fatal("ranged service principals were presented as the complete set")
	}
}

// TestProjectionExcludesManagedSecrets limits state to named metadata even if a malformed SDK observation carries extra attributes.
func TestProjectionExcludesManagedSecrets(t *testing.T) {
	account := accountFixture()
	account.Raw = shared.Object{Attributes: map[string][][]byte{
		"msds-managedpassword": {[]byte("test-only-secret-blob")}, "msds-groupmsamembership": {[]byte("private-descriptor")},
		"description": {[]byte("public description")},
	}}
	values, diagnostics := Project(account)
	if diagnostics.HasError() || values["description"].(types.String).ValueString() != "public description" {
		t.Fatal("named metadata observation failed")
	}
	encoded := fmt.Sprint(values)
	if strings.Contains(encoded, "test-only-secret-blob") || strings.Contains(encoded, "private-descriptor") {
		t.Fatal("unselected secret or security descriptor entered Terraform state")
	}
	for _, forbidden := range []string{"managed_password", "password_wo", "msds_managed_password", "raw_attributes", "password_reader_descriptor", "installed", "host_computers"} {
		if _, found := AttributeTypes()[forbidden]; found {
			t.Fatal("projection advertised password material, host state or independent association ownership")
		}
	}
}

// TestProjectionSeparatesKindsAndEncryption refuses unsupported classes and preserves zero encryption selections.
func TestProjectionSeparatesKindsAndEncryption(t *testing.T) {
	account := accountFixture()
	for _, kind := range []sdkaccounts.ServiceAccountKind{0, 3} {
		account.Kind = kind
		if values, diagnostics := Project(account); !diagnostics.HasError() || values != nil {
			t.Fatal("unsupported managed class emitted a lifecycle identity")
		}
	}
	account.Kind = sdkaccounts.ServiceAccountStandalone
	flags := shared.Flags32(0x20018)
	account.SupportedEncryptionTypes = &flags
	values, diagnostics := Project(account)
	if diagnostics.HasError() || values["kind"].(types.String).ValueString() != "standalone" || len(values["kerberos_encryption_types"].(types.Set).Elements()) != 2 {
		t.Fatal("kind or encryption bits were conflated with unrelated flags")
	}
	flags = 0
	values, diagnostics = Project(account)
	if diagnostics.HasError() || values["kerberos_encryption_types"].IsNull() || len(values["kerberos_encryption_types"].(types.Set).Elements()) != 0 {
		t.Fatal("explicit zero encryption selection became unavailable")
	}
}
