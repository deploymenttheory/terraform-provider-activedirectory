// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package memberships shares observations of direct group links without owning whole member collections.
package memberships

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

var ErrObservation = errors.New("direct group membership observation is incomplete or contradictory")

// Endpoint retains a directory object's stable identity and current observed name.
type Endpoint struct {
	GUID shared.GUID
	DN   string
	Raw  shared.Object
}

// Edge owns one ordered group/member pair; Present is usable only when Confirmed is true.
type Edge struct {
	Group     Endpoint
	Member    Endpoint
	Present   bool
	Confirmed bool
}

// AttributeTypes describes the exact common resource, data-source and list-resource projection.
func AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType, "group_guid": types.StringType, "member_guid": types.StringType,
		"group_dn": types.StringType, "member_dn": types.StringType,
	}
}

// Values returns an importable direct link only after positive confirmation of both endpoint identities and the edge.
func (edge Edge) Values() (map[string]attr.Value, error) {
	if !edge.Confirmed || !edge.Present || edge.Group.DN == "" || edge.Member.DN == "" {
		return nil, ErrObservation
	}
	id, err := identity.RelationshipID(edge.Group.GUID.String(), edge.Member.GUID.String())
	if err != nil {
		return nil, ErrObservation
	}
	return map[string]attr.Value{
		"id": types.StringValue(id), "group_guid": types.StringValue(edge.Group.GUID.String()),
		"member_guid": types.StringValue(edge.Member.GUID.String()), "group_dn": types.StringValue(edge.Group.DN),
		"member_dn": types.StringValue(edge.Member.DN),
	}, nil
}
