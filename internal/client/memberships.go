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

// Memberships exposes direct links and individual membership changes on the qualified provider session.
// Principal resolution supplies stable GUIDs without expanding nested or primary-group membership.
type Memberships interface {
	ResolveIdentity(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Search(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	GroupMemberLinks(context.Context, shared.Identity, memberships.GroupMemberLinksOptions) (memberships.GroupMemberLinksResult, error)
	AddGroupMembers(context.Context, memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error)
	RemoveGroupMembers(context.Context, memberships.RemoveGroupMembersRequest) (memberships.GroupMembershipMutationResult, error)
}

var ErrMemberships = errors.New("configured SDK session does not expose direct group membership operations")

// Memberships returns direct-link operations without creating a connection or changing authentication.
func (b *Bundle) Memberships() (Memberships, error) {
	if b == nil || b.session == nil {
		return nil, ErrMemberships
	}
	getter, ok := b.session.(interface{ MembershipService() Memberships })
	if !ok {
		return nil, ErrMemberships
	}
	service := getter.MembershipService()
	if service == nil {
		return nil, ErrMemberships
	}
	return service, nil
}

// MembershipService combines public SDK membership and identity operations on the existing session.
func (s *sdkSession) MembershipService() Memberships {
	return &sdkMemberships{Service: s.Memberships, session: s}
}

type sdkMemberships struct {
	*memberships.Service
	session *sdkSession
}

// ResolveIdentity reads a direct member's own object identity without following membership links.
func (s *sdkMemberships) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	result, err := s.session.Directory.ResolveIdentity(ctx, target, options)
	if err != nil {
		return result, fmt.Errorf("resolve direct member identity: %w", err)
	}
	return result, nil
}

// Search delegates bounded direct-link assertions to the directory through the public SDK.
func (s *sdkMemberships) Search(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	result, err := s.session.Directory.Search(ctx, options)
	if err != nil {
		return result, fmt.Errorf("assert direct group membership: %w", err)
	}
	return result, nil
}
