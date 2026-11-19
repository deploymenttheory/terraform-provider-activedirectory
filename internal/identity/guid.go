// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package identity defines stable GUID identities for directory objects and individual relationships.
package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var ErrGUID = errors.New("a nonzero Active Directory object GUID is required")

// CanonicalGUID accepts SDK-supported textual GUID forms and returns lowercase canonical identity.
// Names, DNs, SID values and zero GUIDs cannot become Terraform object identities.
func CanonicalGUID(value string) (string, error) {
	guid, err := shared.ParseGUID(value)
	if err != nil || guid == (shared.GUID{}) {
		return "", ErrGUID
	}
	return guid.String(), nil
}

// ImportGUID imports only stable object identity; the next resource Read discovers mutable attributes.
func ImportGUID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := CanonicalGUID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import identity", "Import requires a nonzero Active Directory object GUID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// RelationshipID identifies one directed edge. Relationship type is supplied by the resource type.
func RelationshipID(owner, member string) (string, error) {
	left, err := CanonicalGUID(owner)
	if err != nil {
		return "", err
	}
	right, err := CanonicalGUID(member)
	if err != nil {
		return "", err
	}
	return left + "/" + right, nil
}

// ParseRelationshipID validates and canonicalizes both endpoints without inferring a DN or account name.
func ParseRelationshipID(value string) (owner, member string, diagnostics diag.Diagnostics) {
	left, right, found := strings.Cut(value, "/")
	if found {
		var err error
		owner, err = CanonicalGUID(left)
		if err == nil {
			member, err = CanonicalGUID(right)
		}
		if err == nil {
			return owner, member, nil
		}
	}
	diagnostics.AddError("Invalid relationship identity", "Import requires two nonzero object GUIDs separated by one forward slash, in owner/member order.")
	return "", "", diagnostics
}
