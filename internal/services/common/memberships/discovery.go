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
	var err error
	result.Group, _, err = Resolve(ctx, service, group)
	if err != nil || !hasClass(result.Group.Raw, "group") {
		return result, ErrObservation
	}
	observed, readErr := service.GroupMemberLinks(ctx, group, sdkmemberships.GroupMemberLinksOptions{MaxValues: limit})
	if !discoveryGroupMatches(observed, result.Group) {
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
		matched, err := assertEdge(ctx, service, edge)
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
	if readErr == nil && observed.Complete && len(observed.Members) == 0 {
		result.Complete = false
		return result, ErrVisibility
	}
	if (!observed.AttributeReturned && !result.Complete) || (!observed.Complete && !result.Bounded) {
		readErr = errors.Join(readErr, ErrObservation)
	}
	return result, readErr
}

// discoveryGroupMatches validates the SDK's unique range anchor against independently requested group metadata.
// The range operation does not select distinguishedName; it is never inferred to be a returned attribute.
func discoveryGroupMatches(observed sdkmemberships.GroupMemberLinksResult, group Endpoint) bool {
	if observed.Observation.Object == nil || !observed.Observation.Complete || len(observed.Observation.Candidates) > 1 || observed.Version == nil {
		return false
	}
	if group.GUID != observed.Version.GUID || !hasClass(*observed.Observation.Object, "group") {
		return false
	}
	objects := append([]shared.Object{*observed.Observation.Object}, observed.Observation.Candidates...)
	for _, object := range objects {
		value, err := object.Values("objectGUID").Single()
		guid, parseErr := value.GUID()
		if err != nil || parseErr != nil || guid != group.GUID {
			return false
		}
	}
	return true
}
