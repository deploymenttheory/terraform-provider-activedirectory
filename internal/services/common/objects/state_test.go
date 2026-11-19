// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func observationFixture(t *testing.T) shared.Object {
	t.Helper()
	guid, err := shared.ParseGUID("dc93656c-f745-4926-9bbb-31e7bbecb68b")
	if err != nil {
		t.Fatal(err)
	}
	return shared.Object{DistinguishedName: "cn=Fixture,OU=owned,DC=unit,DC=invalid", Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "name": {[]byte("Fixture")},
		"distinguishedname": {[]byte("CN=Fixture,OU=owned,DC=unit,DC=invalid")},
		"objectclass":       {[]byte("top"), []byte("person"), []byte("organizationalPerson"), []byte("contact")},
	}}
}

// TestProjectPreservesBinaryAndVisibility keeps raw bytes, partial ranges and nullable omission distinct.
func TestProjectPreservesBinaryAndVisibility(t *testing.T) {
	t.Parallel()
	object := observationFixture(t)
	object.RequestedProperties = []string{"jpegPhoto", "description", "member"}
	object.Attributes["jpegphoto"] = [][]byte{{0xff, 0, 0x80}, {}}
	object.Attributes["member;range=0-1499"] = [][]byte{[]byte("CN=Other,DC=unit,DC=invalid")}
	values, diagnostics := Project(object, []string{"jpegPhoto", "description", "member", "telephoneNumber"})
	if diagnostics.HasError() || values["distinguished_name"].(types.String).ValueString() != "CN=Fixture,OU=owned,DC=unit,DC=invalid" {
		t.Fatal("canonical directory-observed DN was not retained")
	}
	if values["object_class"].(types.String).ValueString() != "contact" || len(values["object_classes"].(types.List).Elements()) != 4 {
		t.Fatal("the ordered AD DS class hierarchy was sorted or the instantiated class was guessed")
	}
	raw := values["observed_attributes"].(types.Map).Elements()
	photo := raw["jpegphoto"].(types.Set)
	if photo.IsNull() || len(photo.Elements()) != 2 || !photo.Equal(encodedSet([]shared.RawValue{shared.NewRawValue([]byte{0xff, 0, 0x80}), shared.NewRawValue(nil)})) {
		t.Fatal("binary and zero-length raw values were not retained")
	}
	status := values["attribute_status"].(types.Map).Elements()
	for attribute, expected := range map[string]string{"jpegphoto": "present", "description": "absent", "member": "partial", "telephonenumber": "unloaded"} {
		if status[attribute].(types.String).ValueString() != expected || expected != "present" && !raw[attribute].IsNull() {
			t.Fatal("absent, unloaded or ranged data was fabricated as a complete empty attribute")
		}
	}
	if len(values["attribute_ranges"].(types.Map).Elements()) != 1 {
		t.Fatal("the original range description and its raw values were discarded")
	}
}

// TestProjectionDoesNotPromoteUnusableIdentity rejects zero GUIDs while retaining usable rows with hidden classes.
func TestProjectionDoesNotPromoteUnusableIdentity(t *testing.T) {
	t.Parallel()
	object := observationFixture(t)
	delete(object.Attributes, "objectclass")
	values, diagnostics := Project(object, nil)
	if diagnostics.HasError() || len(diagnostics) == 0 || values["id"].IsNull() || !values["object_class"].IsNull() {
		t.Fatal("a hidden class erased a usable object GUID or invented a class")
	}
	object.Attributes["objectguid"] = [][]byte{make([]byte, 16)}
	if values, diagnostics := Project(object, nil); values != nil || !diagnostics.HasError() {
		t.Fatal("an all-zero GUID became an import identity")
	}
}
