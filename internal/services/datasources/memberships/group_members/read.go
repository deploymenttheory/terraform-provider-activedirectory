// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Read preserves usable partial observations while keeping bounded, hidden and failed discovery distinct from complete absence.
func (d *GroupMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model GroupMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	group, options, diagnostics := constructQuery(model)
	resp.Diagnostics.Append(diagnostics...)
	if d.client == nil {
		resp.Diagnostics.AddError("Group membership client unavailable", "Configure the provider before reading group membership.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	observed, err := d.client.ResolveIdentity(ctx, group, shared.ResolveIdentityOptions{
		Properties: []string{"objectGUID", "objectClass", "distinguishedName"},
	})
	dn, valid := groupMetadata(observed, group)
	if err != nil || !valid {
		resp.Diagnostics.AddError(
			"Group membership scope unavailable",
			"The selected group could not be uniquely observed with the requested identity and metadata. No membership query was performed.",
		)
		return
	}
	result, queryErr := d.client.GroupMembers(ctx, group, options)
	projection := projectResult(result, group, options)
	resp.Diagnostics.Append(projection.diagnostics...)
	projection.complete = projection.complete && queryErr == nil && !resp.Diagnostics.HasError()
	resp.Diagnostics.Append(mapState(ctx, &resp.State, model, group.String(), dn, projection)...)
	switch {
	case queryErr != nil && !onlyLimits(queryErr):
		resp.Diagnostics.AddError(
			"Group membership discovery incomplete",
			"The SDK query failed or returned unresolved observations. Usable rows are retained with complete=false; peer error text is withheld.",
		)
	case projection.bounded:
		resp.Diagnostics.AddWarning(
			"Group membership discovery reached its bound",
			"An aggregate traversal budget stopped retrieval. complete=false; additional matches may or may not exist.",
		)
	case !projection.visibilityKnown:
		resp.Diagnostics.AddWarning(
			"Group membership visibility is indeterminate",
			"At least one visited group returned no direct member values. Empty and hidden attributes are indistinguishable; "+
				"known primary or other observed members are retained with complete=false.",
		)
	case !projection.complete && !resp.Diagnostics.HasError():
		resp.Diagnostics.AddError(
			"Group membership discovery incomplete",
			"Traversal or identity observations did not complete. Returned rows do not establish complete membership or absence.",
		)
	}
}

// groupMetadata checks selected attributes and every positive candidate without mistaking a range locator for a returned DN attribute.
func groupMetadata(result shared.IdentityResult, target shared.Identity) (string, bool) {
	if !result.Complete || result.Object == nil || len(result.Candidates) > 1 {
		return "", false
	}
	dn, valid := objectDN(*result.Object)
	if !valid {
		return "", false
	}
	for _, object := range append([]shared.Object{*result.Object}, result.Candidates...) {
		guid, ok := objectGUID(object)
		candidateDN, complete := objectDN(object)
		if !ok || guid.String() != target.String() || !hasClass(object, "group") || !complete || candidateDN != dn {
			return "", false
		}
	}
	return dn, true
}

// onlyLimits accepts wrappers only when every error leaf is one of the SDK's explicit traversal-budget results.
func onlyLimits(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !onlyLimits(child) {
				return false
			}
		}
		return true
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return onlyLimits(wrapped)
	}
	return errors.Is(err, memberships.ErrMembershipLimit) || errors.Is(err, memberships.ErrGroupMembershipLimit)
}
