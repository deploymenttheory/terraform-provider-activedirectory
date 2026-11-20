// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account_links

import (
	"context"
	"errors"
	"slices"
	"strings"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

// Discovery retains usable direct links independently of range completion and account-resolution failures.
// An omitted account attribute remains an unknown collection, even if the range traversal completed.
type Discovery struct {
	Computer Endpoint
	Edges    []Edge
	Complete bool
	Bounded  bool
}

// Discover follows bounded direct-account ranges and resolves each returned principal to a stable local GUID.
// It performs no local installation, password retrieval or writes.
func Discover(ctx context.Context, service client.ComputerServiceAccountLinks, computer shared.Identity, limit int) (Discovery, error) {
	var result Discovery
	if service == nil || computer.Form() != shared.IdentityFormGUID || limit < 1 || limit > 10000 {
		return result, ErrObservation
	}
	var err error
	result.Computer, _, err = Resolve(ctx, service, computer)
	if err != nil || !hasClass(result.Computer.Raw, "computer") {
		return result, ErrObservation
	}
	observed, readErr := service.ComputerServiceAccounts(ctx, computer, sdkaccounts.ComputerServiceAccountsOptions{MaxValues: limit})
	if !discoveryComputerMatches(observed, result.Computer) {
		return result, errors.Join(readErr, ErrObservation)
	}
	rangeBounded := errors.Is(readErr, sdkaccounts.ErrServiceAccountAssociationLimit)
	projectionFailed := false
	seen := map[shared.GUID]bool{}
	for _, link := range observed.Links {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(readErr, err)
		}
		target, err := shared.IdentityByDN(link.DistinguishedName)
		if err != nil {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		account, _, err := Resolve(ctx, service, target)
		if err != nil || !isManagedAccount(account.Raw) || seen[account.GUID] {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		edge := Edge{Computer: result.Computer, Account: account, Present: true, Confirmed: true}
		matched, err := assertEdge(ctx, service, edge)
		if err != nil || !matched {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		seen[account.GUID] = true
		result.Edges = append(result.Edges, edge)
	}
	slices.SortFunc(result.Edges, func(left, right Edge) int {
		return strings.Compare(left.Account.GUID.String(), right.Account.GUID.String())
	})
	result.Bounded = rangeBounded && !projectionFailed
	result.Complete = readErr == nil && observed.Complete && observed.AttributeReturned
	if readErr == nil && observed.Complete && len(observed.Links) == 0 {
		result.Complete = false
		return result, ErrVisibility
	}
	if (!observed.AttributeReturned && !result.Complete) || (!observed.Complete && !result.Bounded) {
		readErr = errors.Join(readErr, ErrObservation)
	}
	return result, readErr
}

// discoveryComputerMatches validates the SDK's unique range anchor against independently requested computer metadata.
// The range operation does not select distinguishedName; it is never inferred to be a returned attribute.
func discoveryComputerMatches(observed sdkaccounts.ComputerServiceAccountsResult, computer Endpoint) bool {
	if observed.Computer.Object == nil || !observed.Computer.Complete || len(observed.Computer.Candidates) > 1 || observed.Version == nil {
		return false
	}
	if computer.GUID != observed.Version.GUID || !hasClass(*observed.Computer.Object, "computer") {
		return false
	}
	objects := append([]shared.Object{*observed.Computer.Object}, observed.Computer.Candidates...)
	for _, object := range objects {
		value, err := object.Values("objectGUID").Single()
		guid, parseErr := value.GUID()
		if err != nil || parseErr != nil || guid != computer.GUID {
			return false
		}
	}
	return true
}
