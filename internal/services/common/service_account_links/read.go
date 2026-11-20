// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account_links

import (
	"context"
	"errors"
	"strconv"
	"strings"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// Read confirms a direct edge positively, or proves exclusion only with returned account values and a stable computer-version assertion.
// Omitted values, negated filters and an actually empty attribute cannot distinguish absence from denied visibility.
func Read(ctx context.Context, service client.ComputerServiceAccountLinks, computer, account shared.Identity) (Edge, error) {
	var edge Edge
	if service == nil || computer.Form() != shared.IdentityFormGUID || account.Form() != shared.IdentityFormGUID {
		return edge, ErrObservation
	}
	var err error
	var absent bool
	edge.Computer, absent, err = Resolve(ctx, service, computer)
	if err != nil {
		edge.Confirmed = absent
		return edge, err
	}
	if !hasClass(edge.Computer.Raw, "computer") {
		return edge, ErrObservation
	}
	edge.Account, absent, err = Resolve(ctx, service, account)
	if err != nil {
		edge.Confirmed = absent
		return edge, err
	}
	if !isManagedAccount(edge.Account.Raw) {
		return edge, ErrObservation
	}
	matched, err := assertEdge(ctx, service, edge)
	if err != nil {
		return edge, err
	}
	if !matched {
		if err := confirmExclusion(ctx, service, edge, computer); err != nil {
			return edge, err
		}
	}
	current, _, err := Resolve(ctx, service, account)
	if err != nil || current.GUID != edge.Account.GUID || current.DN != edge.Account.DN {
		return edge, ErrObservation
	}
	edge.Present, edge.Confirmed = matched, true
	return edge, nil
}

// confirmExclusion uses one returned value as a read-visibility witness, then pins the negative assertion to that exact GUID/USN.
// A missing attribute never becomes a witness. Reading one value avoids enumerating unrelated accounts for an individual edge.
func confirmExclusion(ctx context.Context, service client.ComputerServiceAccountLinks, edge Edge, computer shared.Identity) error {
	witness, err := service.ComputerServiceAccounts(ctx, computer, sdkaccounts.ComputerServiceAccountsOptions{MaxValues: 1, MaxRanges: 1})
	if err != nil && !errors.Is(err, sdkaccounts.ErrServiceAccountAssociationLimit) {
		return ErrObservation
	}
	if !discoveryComputerMatches(witness, edge.Computer) {
		return ErrObservation
	}
	if !witness.AttributeReturned || len(witness.Links) == 0 {
		return ErrVisibility
	}
	if len(witness.Ranges) != 1 || witness.Ranges[0].Error != nil {
		return ErrObservation
	}
	assertion := shared.FilterAnd(
		shared.FilterEqual("uSNChanged", shared.FilterText(strconv.FormatUint(witness.Version.USNChanged, 10))),
		shared.FilterNot(shared.FilterEqual("msDS-HostServiceAccount", shared.FilterText(edge.Account.DN))),
	)
	matched, readErr := assertComputer(ctx, service, edge.Computer, assertion)
	if readErr != nil || !matched {
		return ErrObservation
	}
	return nil
}

// Resolve requires a complete unique SDK result and agreement with the requested GUID where supplied.
// Positive candidates prevent a mixed or contradictory not-found response from establishing absence.
func Resolve(ctx context.Context, service client.ComputerServiceAccountLinks, target shared.Identity) (Endpoint, bool, error) {
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

// assertEdge asks the server to return this exact computer when the selected direct-link assertion is true.
func assertEdge(ctx context.Context, service client.ComputerServiceAccountLinks, edge Edge) (bool, error) {
	account := shared.FilterEqual("msDS-HostServiceAccount", shared.FilterText(edge.Account.DN))
	return assertComputer(ctx, service, edge.Computer, account)
}

// assertComputer accepts only a positively returned, matching computer; zero rows are never a negative proof.
func assertComputer(ctx context.Context, service client.ComputerServiceAccountLinks, computer Endpoint, assertion shared.Filter) (bool, error) {
	result, err := service.Search(ctx, shared.SearchOptions{
		Base: computer.DN, Scope: shared.ScopeBase, PageSize: 2, ResultLimit: 2,
		TypedFilter: shared.FilterAnd(
			shared.FilterEqual("objectClass", shared.FilterText("computer")),
			shared.FilterEqual("objectGUID", shared.FilterBytes(computer.GUID.Bytes())), assertion,
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
	if err != nil || observed.GUID != computer.GUID || observed.DN != computer.DN || !hasClass(observed.Raw, "computer") {
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

// isManagedAccount accepts only standalone and group managed service-account classes.
func isManagedAccount(object shared.Object) bool {
	return hasClass(object, "msDS-ManagedServiceAccount") || hasClass(object, "msDS-GroupManagedServiceAccount")
}
