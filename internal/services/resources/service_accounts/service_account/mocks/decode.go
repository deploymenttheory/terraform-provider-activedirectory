// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// DecodeServiceAccountFixture projects copied public SDK observations from a feature-local response fixture.
func DecodeServiceAccountFixture(payload []byte) (service_accounts.ServiceAccount, error) {
	var fixture struct {
		GUID        string  `json:"guid"`
		Name        string  `json:"name"`
		DN          string  `json:"distinguishedName"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		return service_accounts.ServiceAccount{}, fmt.Errorf("decode service_account fixture: %w", err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		return service_accounts.ServiceAccount{}, fmt.Errorf("decode service_account fixture GUID: %w", err)
	}
	attributes := map[string][][]byte{
		"objectguid":     {guid.Bytes()},
		"name":           {[]byte(fixture.Name)},
		"objectclass":    {[]byte("msDS-GroupManagedServiceAccount")},
		"samaccountname": {[]byte(fixture.Name + "$")},
	}
	if fixture.Description != nil {
		attributes["description"] = [][]byte{[]byte(*fixture.Description)}
	}
	return service_accounts.ServiceAccount{
		GUID: &guid, Name: &fixture.Name, DistinguishedName: fixture.DN,
		Kind: service_accounts.ServiceAccountGroupManaged, Description: fixture.Description,
		AccountControl: accounts.AccountControlState{Enabled: fixture.Enabled},
		Raw:            shared.Object{DistinguishedName: fixture.DN, Attributes: attributes, RequestedProperties: []string{"description"}},
	}, nil
}
