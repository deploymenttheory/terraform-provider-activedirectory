// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

// Factory supplies an explicitly synthetic portable TLS fixture session for isolated OU feature factories.
func Factory(units client.OrganizationalUnits) client.Factory {
	return sdkmocks.Factory(func() client.Session {
		return &Session{Units: units, Session: &sdkmocks.Session{
			RootDSEFunc: func(context.Context, shared.RootDSEOptions) (shared.RootDSE, error) {
				partition := "DC=unit,DC=invalid"
				return shared.RootDSE{DefaultNamingContext: &partition, SupportedCapabilities: []string{"1.2.840.113556.1.4.800"}}, nil
			},
			ProtectionFunc: func(context.Context) (activedirectory.SessionProtection, error) {
				return activedirectory.SessionProtection{
					Backend: config.BackendPortable, Authentication: config.AuthenticationSimple,
					Transport: config.TransportLDAPS, Complete: true, TLSVerified: true, TLSVersion: tls.VersionTLS13,
					TLSCipherSuite: tls.TLS_AES_256_GCM_SHA384, CipherBits: 256,
				}, nil
			},
		}}
	})
}

// DecodeUnitFixture builds copied public SDK observations from a feature-local synthetic response fixture.
func DecodeUnitFixture(payload []byte) (organizational_units.OrganizationalUnit, error) {
	var fixture struct {
		GUID        string  `json:"guid"`
		Name        string  `json:"name"`
		DN          string  `json:"distinguishedName"`
		Description *string `json:"description"`
		Protected   *bool   `json:"protectedFromAccidentalDeletion"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		return organizational_units.OrganizationalUnit{}, fmt.Errorf("decode OU fixture: %w", err)
	}
	guid, err := shared.ParseGUID(fixture.GUID)
	if err != nil {
		return organizational_units.OrganizationalUnit{}, fmt.Errorf("decode OU fixture GUID: %w", err)
	}
	attributes := map[string][][]byte{"objectguid": {guid.Bytes()}, "name": {[]byte(fixture.Name)}, "objectclass": {[]byte("organizationalUnit")}}
	if fixture.Description != nil {
		attributes["description"] = [][]byte{[]byte(*fixture.Description)}
	}
	return organizational_units.OrganizationalUnit{
		GUID: &guid, Name: &fixture.Name, DistinguishedName: fixture.DN,
		Description: fixture.Description, ProtectedFromAccidentalDeletion: fixture.Protected,
		Raw: shared.Object{DistinguishedName: fixture.DN, Attributes: attributes, RequestedProperties: []string{"description"}},
	}, nil
}
