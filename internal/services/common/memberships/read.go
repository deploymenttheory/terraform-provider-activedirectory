// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships

import (
	"context"
	"strings"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Read confirms a direct edge with a positive server assertion, never with an omitted member attribute.
// A second, explicit negated assertion can confirm nonmembership, including an actually empty group.
// Zero rows, failed searches and unstable endpoint names remain indeterminate rather than removing state.
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
	for _, present := range []bool{true, false} {
		matched, err := assertEdge(ctx, service, edge, present)
		if err != nil {
			return edge, err
		}
		if !matched {
			continue
		}
		current, _, err := Resolve(ctx, service, member)
		if err != nil || current.GUID != edge.Member.GUID || current.DN != edge.Member.DN {
			return edge, ErrObservation
		}
		edge.Present, edge.Confirmed = present, true
		return edge, nil
	}
	return edge, ErrObservation
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
func assertEdge(ctx context.Context, service client.Memberships, edge Edge, present bool) (bool, error) {
	member := shared.FilterEqual("member", shared.FilterText(edge.Member.DN))
	if !present {
		member = shared.FilterNot(member)
	}
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
