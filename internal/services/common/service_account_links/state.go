// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package service_account_links shares observations of direct computer links without owning whole account collections.
package service_account_links

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

var ErrObservation = errors.New("direct computer association observation is incomplete or contradictory")

var ErrVisibility = errors.New("direct account values cannot distinguish absence from hidden association")

// Endpoint retains a directory object's stable identity and current observed name.
type Endpoint struct {
	GUID shared.GUID
	DN   string
	Raw  shared.Object
}

// Edge owns one ordered computer/account pair; Present is usable only when Confirmed is true.
type Edge struct {
	Computer  Endpoint
	Account   Endpoint
	Present   bool
	Confirmed bool
}

// AttributeTypes describes the exact common resource, data-source and list-resource projection.
func AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType, "computer_guid": types.StringType, "service_account_guid": types.StringType,
		"computer_dn": types.StringType, "service_account_dn": types.StringType,
	}
}

// Values returns an importable direct link only after positive confirmation of both endpoint identities and the edge.
func (edge Edge) Values() (map[string]attr.Value, error) {
	if !edge.Confirmed || !edge.Present || edge.Computer.DN == "" || edge.Account.DN == "" {
		return nil, ErrObservation
	}
	id, err := identity.RelationshipID(edge.Computer.GUID.String(), edge.Account.GUID.String())
	if err != nil {
		return nil, ErrObservation
	}
	return map[string]attr.Value{
		"id": types.StringValue(id), "computer_guid": types.StringValue(edge.Computer.GUID.String()),
		"service_account_guid": types.StringValue(edge.Account.GUID.String()), "computer_dn": types.StringValue(edge.Computer.DN),
		"service_account_dn": types.StringValue(edge.Account.DN),
	}, nil
}
