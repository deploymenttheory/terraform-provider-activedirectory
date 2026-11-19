// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// DecodeObjectFixture reads explicit synthetic SDK observations; JSON binary attribute values use base64.
func DecodeObjectFixture(payload []byte) (shared.Object, error) {
	var fixture struct {
		GUID       string              `json:"guid"`
		Name       string              `json:"name"`
		DN         string              `json:"dn"`
		Classes    []string            `json:"classes"`
		Attributes map[string][][]byte `json:"attributes"`
		Properties []string            `json:"properties"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		return shared.Object{}, fmt.Errorf("decode synthetic object fixture: %w", err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		return shared.Object{}, fmt.Errorf("decode synthetic object GUID: %w", err)
	}
	object := shared.Object{DistinguishedName: fixture.DN, RequestedProperties: fixture.Properties, Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "name": {[]byte(fixture.Name)}, "distinguishedname": {[]byte(fixture.DN)},
	}}
	for _, class := range fixture.Classes {
		object.Attributes["objectclass"] = append(object.Attributes["objectclass"], []byte(class))
	}
	for name, values := range fixture.Attributes {
		object.Attributes[strings.ToLower(name)] = values
	}
	return object, nil
}
