// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships

import (
	"context"
	"errors"
	"strconv"
	"strings"

	sdkmemberships "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Read confirms a direct edge positively, or proves exclusion only with returned member values and a stable group-version assertion.
// Omitted values, negated filters and an actually empty attribute cannot distinguish absence from denied visibility.
func Read(ctx context.Context, service client.Memberships, group, member shared.Identity) (Edge, error) {
	var edge Edge
	if service == nil || group.Form() != shared.IdentityFormGUID || member.Form() != shared.IdentityFormGUID {
		return edge, ErrObservation
	}
	var err error
	var absent bool
	edge.Group, absent, err = Resolve(ctx, service, group)
	if err != nil {
		edge.Confirmed = absent
		return edge, err
	}
	if !hasClass(edge.Group.Raw, "group") {
		return edge, ErrObservation
	}
	edge.Member, absent, err = Resolve(ctx, service, member)
	if err != nil {
		edge.Confirmed = absent
		return edge, err
	}
	if hasClass(edge.Member.Raw, "foreignSecurityPrincipal") {
		return edge, ErrObservation
	}
	matched, err := assertEdge(ctx, service, edge)
	if err != nil {
		return edge, err
	}
	if !matched {
		if err := confirmExclusion(ctx, service, edge, group); err != nil {
			return edge, err
		}
	}
	current, _, err := Resolve(ctx, service, member)
	if err != nil || current.GUID != edge.Member.GUID || current.DN != edge.Member.DN {
		return edge, ErrObservation
	}
	edge.Present, edge.Confirmed = matched, true
	return edge, nil
}

// confirmExclusion uses one returned value as a read-visibility witness, then pins the negative assertion to that exact GUID/USN.
// A missing attribute never becomes a witness. Reading one value avoids enumerating unrelated members for an individual edge.
func confirmExclusion(ctx context.Context, service client.Memberships, edge Edge, group shared.Identity) error {
	witness, err := service.GroupMemberLinks(ctx, group, sdkmemberships.GroupMemberLinksOptions{MaxValues: 1, MaxRanges: 1})
	if err != nil && !errors.Is(err, sdkmemberships.ErrGroupMembershipLimit) {
		return ErrObservation
	}
	if !discoveryGroupMatches(witness, edge.Group) {
		return ErrObservation
	}
	if !witness.AttributeReturned || len(witness.Members) == 0 {
		return ErrVisibility
	}
	if len(witness.Ranges) != 1 || witness.Ranges[0].Error != nil {
		return ErrObservation
	}
	assertion := shared.FilterAnd(
		shared.FilterEqual("uSNChanged", shared.FilterText(strconv.FormatUint(witness.Version.USNChanged, 10))),
		shared.FilterNot(shared.FilterEqual("member", shared.FilterText(edge.Member.DN))),
	)
	matched, readErr := assertGroup(ctx, service, edge.Group, assertion)
	if readErr != nil || !matched {
		return ErrObservation
	}
	return nil
}

// Resolve requires a complete unique SDK result and agreement with the requested GUID where supplied.
// Positive candidates prevent a mixed or contradictory not-found response from establishing absence.
func Resolve(ctx context.Context, service client.Memberships, target shared.Identity) (Endpoint, bool, error) {
	result, err := service.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{
		Properties: []string{"objectGUID", "distinguishedName", "objectClass"},
	})
	if err != nil {
		return Endpoint{}, lifecycle.ConfirmedAbsent(result, err), err
	}
	if !result.Complete || result.Object == nil || len(result.Candidates) > 1 {
		return Endpoint{}, false, ErrObservation
	}
	endpoint, err := projectEndpoint(*result.Object)
	if err != nil || (target.Form() == shared.IdentityFormGUID && endpoint.GUID.String() != target.String()) {
		return Endpoint{}, false, ErrObservation
	}
	for _, candidate := range result.Candidates {
		other, err := projectEndpoint(candidate)
		if err != nil || other.GUID != endpoint.GUID {
			return Endpoint{}, false, ErrObservation
		}
	}
	return endpoint, false, nil
}

// assertEdge asks the server to return this exact group when the selected direct-link assertion is true.
func assertEdge(ctx context.Context, service client.Memberships, edge Edge) (bool, error) {
	member := shared.FilterEqual("member", shared.FilterText(edge.Member.DN))
	return assertGroup(ctx, service, edge.Group, member)
}

// assertGroup accepts only a positively returned, matching group; zero rows are never a negative proof.
func assertGroup(ctx context.Context, service client.Memberships, group Endpoint, assertion shared.Filter) (bool, error) {
	result, err := service.Search(ctx, shared.SearchOptions{
		Base: group.DN, Scope: shared.ScopeBase, PageSize: 2, ResultLimit: 2,
		TypedFilter: shared.FilterAnd(
			shared.FilterEqual("objectClass", shared.FilterText("group")),
			shared.FilterEqual("objectGUID", shared.FilterBytes(group.GUID.Bytes())), assertion,
		),
		Properties: []string{"objectGUID", "distinguishedName", "objectClass"},
	})
	if err != nil || !result.Complete || result.LimitReached || len(result.Objects) > 1 {
		return false, ErrObservation
	}
	if len(result.Objects) == 0 {
		return false, nil
	}
	observed, err := projectEndpoint(result.Objects[0])
	if err != nil || observed.GUID != group.GUID || observed.DN != group.DN || !hasClass(observed.Raw, "group") {
		return false, ErrObservation
	}
	return true, nil
}

// projectEndpoint uses returned directory attributes without interpreting a mutable name as stable identity.
func projectEndpoint(object shared.Object) (Endpoint, error) {
	value, err := object.Values("objectGUID").Single()
	guid, guidErr := value.GUID()
	name, nameErr := object.Values("distinguishedName").Single()
	dn, textErr := name.Text()
	if err != nil || guidErr != nil || nameErr != nil || textErr != nil || guid == (shared.GUID{}) {
		return Endpoint{}, ErrObservation
	}
	if _, err := shared.IdentityByDN(dn); err != nil {
		return Endpoint{}, ErrObservation
	}
	return Endpoint{GUID: guid, DN: dn, Raw: object}, nil
}

func hasClass(object shared.Object, class string) bool {
	values := object.Values("objectClass")
	if values.State != shared.AttributePresent {
		return false
	}
	for _, value := range values.Values {
		text, err := value.Text()
		if err == nil && strings.EqualFold(text, class) {
			return true
		}
	}
	return false
}
