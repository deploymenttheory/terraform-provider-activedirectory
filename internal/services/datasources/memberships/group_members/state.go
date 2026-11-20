// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type memberRow struct {
	key, sid, dn, class string
	values              map[string]attr.Value
	primary, invalid    bool
}

type projectedResult struct {
	members, unresolved []attr.Value
	complete, bounded   bool
	visibilityKnown     bool
	groups              int
	diagnostics         diag.Diagnostics
}

// projectResult validates traversal anchors and principal identities before publishing deterministic, bounded partial rows.
func projectResult(result memberships.GroupMembersResult, group shared.Identity, options memberships.GroupMembersOptions) projectedResult {
	projection := projectedResult{complete: result.Complete, bounded: result.LimitReached, visibilityKnown: len(result.Groups) > 0, groups: len(result.Groups)}
	if len(result.Groups) == 0 || len(result.Groups) > options.Limits.MaxGroups {
		projection.diagnostics.AddError("Invalid membership traversal", "The SDK returned no group anchor or exceeded the selected group budget.")
	}
	for index, traversal := range result.Groups {
		if !validTraversal(traversal, group, index == 0) || traversal.Depth > options.Limits.MaxDepth || (!options.Recursive && (index != 0 || traversal.Depth != 0)) {
			projection.diagnostics.AddError("Invalid membership traversal identity", "A returned group version or candidate identity is incomplete or contradictory.")
		}
		if !traversal.Links.AttributeReturned || len(traversal.Links.Members) == 0 {
			projection.visibilityKnown = false
		}
		projection.complete = projection.complete && traversal.Links.Complete
	}
	if projection.diagnostics.HasError() {
		projection.complete = false
		return projection
	}
	for _, traversal := range result.Groups {
		for _, observedRange := range traversal.Links.Ranges {
			if observedRange.Error != nil {
				projection.diagnostics.AddError(
					"Membership range observation failed",
					"A returned range carries an error. Usable independently identified principals are retained; complete=false.",
				)
			}
		}
		for _, resolution := range traversal.Resolutions {
			if resolution.Error != nil {
				projection.diagnostics.AddError(
					"Membership principal resolution failed",
					"A traversal reference could not be resolved. Peer error text is withheld and complete=false.",
				)
			}
		}
	}
	projection.projectMembers(result.Members, options.Limits.MaxObjects)
	projection.projectUnresolved(result.Unresolved, options.Limits.MaxObjects)
	projection.complete = projection.complete && !projection.bounded && projection.visibilityKnown && len(result.Unresolved) == 0 && !projection.diagnostics.HasError()
	return projection
}

// validTraversal verifies GUID/class candidates against the SDK version without requiring an unselected distinguishedName attribute.
func validTraversal(node memberships.GroupMembershipTraversal, root shared.Identity, first bool) bool {
	links := node.Links
	if links.Version == nil || links.Version.GUID == (shared.GUID{}) || !links.Observation.Complete ||
		links.Observation.Object == nil || len(links.Observation.Candidates) > 1 || node.Depth < 0 {
		return false
	}
	if first && (links.Version.GUID.String() != root.String() || node.Depth != 0) {
		return false
	}
	for _, object := range append([]shared.Object{*links.Observation.Object}, links.Observation.Candidates...) {
		guid, ok := objectGUID(object)
		if !ok || guid != links.Version.GUID || !hasClass(object, "group") {
			return false
		}
	}
	return true
}

// projectMembers keeps only GUID/SID-consistent principals, merges agreeing duplicate paths and rejects conflicting duplicate metadata.
func (projection *projectedResult) projectMembers(principals []memberships.MembershipPrincipal, limit int) {
	rows := map[string]*memberRow{}
	sids := map[string]*memberRow{}
	for _, principal := range principals {
		row, valid := projectMember(principal)
		if !valid {
			projection.diagnostics.AddError(
				"Incomplete group member identity",
				"A returned principal lacks consistent GUID, SID, source, class or explicitly requested name metadata.",
			)
			continue
		}
		if previous, duplicate := rows[row.key]; duplicate {
			if previous.sid != row.sid || previous.dn != row.dn || previous.class != row.class || !sameNames(previous, row) {
				previous.invalid = true
				projection.diagnostics.AddError(
					"Conflicting group member identity",
					"Duplicate GUID observations disagree. That principal is omitted from usable rows; complete=false.",
				)
			}
			previous.primary = previous.primary || row.primary
			continue
		}
		if previous, duplicate := sids[row.sid]; duplicate && previous.key != row.key {
			previous.invalid, row.invalid = true, true
			projection.diagnostics.AddError(
				"Conflicting group member SID",
				"The same security identifier identifies different observed objects. Neither is a usable resolved principal.",
			)
		}
		if len(rows) >= limit {
			projection.bounded = true
			break
		}
		rows[row.key], sids[row.sid] = &row, &row
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		row := rows[key]
		if row.invalid {
			continue
		}
		row.values["primary"] = types.BoolValue(row.primary)
		value, diagnostics := types.ObjectValue(memberTypes(), row.values)
		projection.diagnostics.Append(diagnostics...)
		projection.members = append(projection.members, value)
	}
}

// projectUnresolved retains bounded valid directory references without manufacturing a principal or exposing SDK error text.
func (projection *projectedResult) projectUnresolved(unresolved []memberships.MembershipResolution, limit int) {
	rows := map[string]attr.Value{}
	for _, reference := range unresolved {
		if _, err := shared.IdentityByDN(reference.Reference); err != nil {
			projection.diagnostics.AddError("Invalid unresolved membership reference", "The SDK returned an invalid directory reference; its raw contents are withheld.")
			continue
		}
		key := strings.ToLower(reference.Source.Host) + "/" + reference.Reference
		if _, duplicate := rows[key]; duplicate {
			continue
		}
		if len(rows) >= limit {
			projection.bounded = true
			break
		}
		value, diagnostics := types.ObjectValue(unresolvedTypes(), map[string]attr.Value{
			"reference": types.StringValue(reference.Reference), "source_host": types.StringValue(reference.Source.Host),
		})
		projection.diagnostics.Append(diagnostics...)
		rows[key] = value
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		projection.unresolved = append(projection.unresolved, rows[key])
	}
}

// mapState preserves the caller's optional inputs and writes only validated observations and explicit completeness flags.
func mapState(ctx context.Context, state *tfsdk.State, model GroupMembersDataSourceModel, guid, dn string, result projectedResult) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	model.ID, model.GroupDN = types.StringValue(guid), types.StringValue(dn)
	model.Complete, model.LimitReached = types.BoolValue(result.complete), types.BoolValue(result.bounded)
	model.GroupsVisited, model.VisibilityKnown = types.Int64Value(int64(result.groups)), types.BoolValue(result.visibilityKnown)
	var conversion diag.Diagnostics
	model.Members, conversion = types.ListValue(types.ObjectType{AttrTypes: memberTypes()}, result.members)
	diagnostics.Append(conversion...)
	model.Unresolved, conversion = types.ListValue(types.ObjectType{AttrTypes: unresolvedTypes()}, result.unresolved)
	diagnostics.Append(conversion...)
	diagnostics.Append(state.Set(ctx, &model)...)
	return diagnostics
}

// projectMember requires returned GUID/SID attributes to agree with the SDK principal, retaining no unresolved FSP as an account identity.
func projectMember(principal memberships.MembershipPrincipal) (memberRow, bool) {
	row := memberRow{primary: principal.Primary, values: map[string]attr.Value{}}
	guid, guidOK := objectGUID(principal.Object)
	dn, dnOK := objectDN(principal.Object)
	sidValue, sidErr := principal.Object.Values("objectSid").Single()
	sid, parseErr := sidValue.SID()
	classes := principal.Object.Values("objectClass")
	if !guidOK || guid != principal.GUID || !dnOK || sidErr != nil || parseErr != nil || sid.String() != principal.SID.String() ||
		principal.Source.Host == "" || classes.State != shared.AttributePresent || len(classes.Values) == 0 || hasClass(principal.Object, "foreignSecurityPrincipal") {
		return row, false
	}
	class, classErr := classes.Values[len(classes.Values)-1].Text()
	if classErr != nil || class == "" {
		return row, false
	}
	row.key = strings.ToLower(principal.Source.Host) + ":" + strconv.Itoa(int(principal.Source.Port)) + "/" + guid.String()
	row.sid, row.dn, row.class = sid.String(), dn, class
	row.values = map[string]attr.Value{
		"id": types.StringValue(guid.String()), "sid": types.StringValue(row.sid), "distinguished_name": types.StringValue(dn),
		"object_class": types.StringValue(class), "source_host": types.StringValue(principal.Source.Host),
		"source_port": types.Int64Value(int64(principal.Source.Port)), "primary": types.BoolValue(principal.Primary),
	}
	for field, property := range map[string]string{"name": "name", "sam_account_name": "sAMAccountName"} {
		row.values[field] = types.StringNull()
		values := principal.Object.Values(property)
		if values.State != shared.AttributePresent {
			continue
		}
		value, err := values.Single()
		text, textErr := value.Text()
		if err != nil || textErr != nil {
			return row, false
		}
		row.values[field] = types.StringValue(text)
	}
	return row, true
}

func sameNames(left *memberRow, right memberRow) bool {
	return left.values["name"].Equal(right.values["name"]) && left.values["sam_account_name"].Equal(right.values["sam_account_name"])
}

func objectGUID(object shared.Object) (shared.GUID, bool) {
	value, err := object.Values("objectGUID").Single()
	guid, parseErr := value.GUID()
	return guid, err == nil && parseErr == nil && guid != (shared.GUID{})
}

func objectDN(object shared.Object) (string, bool) {
	value, err := object.Values("distinguishedName").Single()
	dn, textErr := value.Text()
	if err != nil || textErr != nil {
		return "", false
	}
	_, parseErr := shared.IdentityByDN(dn)
	return dn, parseErr == nil
}

func hasClass(object shared.Object, class string) bool {
	for _, value := range object.Values("objectClass").Values {
		text, err := value.Text()
		if err == nil && strings.EqualFold(text, class) {
			return true
		}
	}
	return false
}

func memberTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType, "sid": types.StringType, "distinguished_name": types.StringType, "name": types.StringType,
		"sam_account_name": types.StringType, "object_class": types.StringType, "source_host": types.StringType,
		"source_port": types.Int64Type, "primary": types.BoolType,
	}
}

func unresolvedTypes() map[string]attr.Type {
	return map[string]attr.Type{"reference": types.StringType, "source_host": types.StringType}
}
