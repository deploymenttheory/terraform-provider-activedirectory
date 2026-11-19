// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package memberships

import (
	"context"
	"errors"
	"slices"
	"strings"

	sdkmemberships "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

// Discovery retains usable direct links independently of range completion and member-resolution failures.
// An omitted member attribute remains an unknown collection, even if the range traversal completed.
type Discovery struct {
	Group    Endpoint
	Edges    []Edge
	Complete bool
	Bounded  bool
}

// Discover follows bounded direct-member ranges and resolves each returned principal to a stable local GUID.
// It performs no recursive expansion, primary-group inference, foreign-principal resolution or writes.
func Discover(ctx context.Context, service client.Memberships, group shared.Identity, limit int) (Discovery, error) {
	var result Discovery
	if service == nil || group.Form() != shared.IdentityFormGUID || limit < 1 || limit > 10000 {
		return result, ErrObservation
	}
	observed, readErr := service.GroupMemberLinks(ctx, group, sdkmemberships.GroupMemberLinksOptions{MaxValues: limit})
	var err error
	result.Group, err = discoveryGroup(observed, group)
	if err != nil {
		return result, errors.Join(readErr, ErrObservation)
	}
	rangeBounded := errors.Is(readErr, sdkmemberships.ErrGroupMembershipLimit)
	projectionFailed := false
	seen := map[shared.GUID]bool{}
	for _, link := range observed.Members {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(readErr, err)
		}
		target, err := shared.IdentityByDN(link.DistinguishedName)
		if err != nil {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		member, _, err := Resolve(ctx, service, target)
		if err != nil || hasClass(member.Raw, "foreignSecurityPrincipal") || seen[member.GUID] {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		edge := Edge{Group: result.Group, Member: member, Present: true, Confirmed: true}
		matched, err := assertEdge(ctx, service, edge, true)
		if err != nil || !matched {
			readErr = errors.Join(readErr, ErrObservation)
			projectionFailed = true
			continue
		}
		seen[member.GUID] = true
		result.Edges = append(result.Edges, edge)
	}
	slices.SortFunc(result.Edges, func(left, right Edge) int {
		return strings.Compare(left.Member.GUID.String(), right.Member.GUID.String())
	})
	result.Bounded = rangeBounded && !projectionFailed
	result.Complete = readErr == nil && observed.Complete && observed.AttributeReturned
	if !observed.AttributeReturned && readErr == nil && observed.Complete && len(observed.Members) == 0 {
		empty, err := assertGroup(ctx, service, result.Group, shared.FilterNot(shared.FilterPresent("member")))
		result.Complete = err == nil && empty
	}
	if (!observed.AttributeReturned && !result.Complete) || (!observed.Complete && !result.Bounded) {
		readErr = errors.Join(readErr, ErrObservation)
	}
	return result, readErr
}

// discoveryGroup requires unique group identity and agreement with the SDK's stable range-version anchor.
func discoveryGroup(observed sdkmemberships.GroupMemberLinksResult, target shared.Identity) (Endpoint, error) {
	if observed.Observation.Object == nil || !observed.Observation.Complete || len(observed.Observation.Candidates) > 1 || observed.Version == nil {
		return Endpoint{}, ErrObservation
	}
	group, err := projectEndpoint(*observed.Observation.Object)
	if err != nil || group.GUID.String() != target.String() || group.GUID != observed.Version.GUID || !hasClass(group.Raw, "group") {
		return Endpoint{}, ErrObservation
	}
	for _, candidate := range observed.Observation.Candidates {
		other, err := projectEndpoint(candidate)
		if err != nil || other.GUID != group.GUID {
			return Endpoint{}, ErrObservation
		}
	}
	return group, nil
}
