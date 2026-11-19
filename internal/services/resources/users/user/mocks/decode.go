// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// DecodeUserFixture projects copied public SDK observations from a feature-local response fixture.
func DecodeUserFixture(payload []byte) (users.User, error) {
	var fixture struct {
		GUID        string  `json:"guid"`
		Name        string  `json:"name"`
		DN          string  `json:"distinguishedName"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		return users.User{}, fmt.Errorf("decode user fixture: %w", err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		return users.User{}, fmt.Errorf("decode user fixture GUID: %w", err)
	}
	attributes := map[string][][]byte{"objectguid": {guid.Bytes()}, "name": {[]byte(fixture.Name)}, "objectclass": {[]byte("user")}, "samaccountname": {[]byte(fixture.Name)}}
	if fixture.Description != nil {
		attributes["description"] = [][]byte{[]byte(*fixture.Description)}
	}
	return users.User{
		GUID: &guid, Name: &fixture.Name, DistinguishedName: fixture.DN,
		UserProperties: users.UserProperties{SAMAccountName: &fixture.Name, Description: fixture.Description, Enabled: fixture.Enabled},
		Raw:            shared.Object{DistinguishedName: fixture.DN, Attributes: attributes, RequestedProperties: []string{"description"}},
	}, nil
}
