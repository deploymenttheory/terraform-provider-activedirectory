// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// DecodeGroupFixture constructs a synthetic public SDK observation; it performs no directory lookup or schema inference.
func DecodeGroupFixture(data []byte) (groups.Group, error) {
	var fixture struct {
		GUID           string `json:"guid"`
		Name           string `json:"name"`
		DN             string `json:"dn"`
		Description    string `json:"description"`
		SAMAccountName string `json:"samAccountName"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		return groups.Group{}, fmt.Errorf("decode synthetic group fixture: %w", err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		return groups.Group{}, fmt.Errorf("parse synthetic group GUID: %w", err)
	}
	category, scope := groups.GroupSecurity, groups.GroupGlobal
	return groups.Group{
		GUID: &guid, Name: &fixture.Name, DistinguishedName: fixture.DN, SAMAccountName: &fixture.SAMAccountName,
		Description: &fixture.Description, Category: &category, Scope: &scope,
		Raw: shared.Object{
			DistinguishedName: fixture.DN, RequestedProperties: []string{"sAMAccountName", "description", "displayName", "wWWHomePage", "managedBy"},
			Attributes: map[string][][]byte{
				"objectguid": {guid.Bytes()}, "objectclass": {[]byte("top"), []byte("group")}, "name": {[]byte(fixture.Name)},
				"samaccountname": {[]byte(fixture.SAMAccountName)}, "description": {[]byte(fixture.Description)},
			},
		},
	}, nil
}
