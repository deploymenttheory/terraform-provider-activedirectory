// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// GroupMembershipQueries reads direct or recursive membership on the provider's qualified session.
type GroupMembershipQueries interface {
	ResolveIdentity(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	GroupMembers(context.Context, shared.Identity, memberships.GroupMembersOptions) (memberships.GroupMembersResult, error)
}

var ErrGroupMembershipQueries = errors.New("configured SDK session does not expose group membership queries")

// GroupMembershipQueries returns read-only operations without opening another session or following foreign endpoints.
func (b *Bundle) GroupMembershipQueries() (GroupMembershipQueries, error) {
	if b == nil || b.session == nil {
		return nil, ErrGroupMembershipQueries
	}
	getter, ok := b.session.(interface{ GroupMembershipQueryService() GroupMembershipQueries })
	if !ok {
		return nil, ErrGroupMembershipQueries
	}
	service := getter.GroupMembershipQueryService()
	if service == nil {
		return nil, ErrGroupMembershipQueries
	}
	return service, nil
}

// GroupMembershipQueryService composes only public SDK reads on the existing connection and lifetime.
func (s *sdkSession) GroupMembershipQueryService() GroupMembershipQueries {
	return &sdkGroupMembershipQueries{Service: s.Memberships, session: s}
}

type sdkGroupMembershipQueries struct {
	*memberships.Service
	session *sdkSession
}

// ResolveIdentity explicitly requests the group's stable identity and current directory metadata.
func (s *sdkGroupMembershipQueries) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	result, err := s.session.Directory.ResolveIdentity(ctx, target, options)
	if err != nil {
		return result, fmt.Errorf("resolve membership query group: %w", err)
	}
	return result, nil
}
