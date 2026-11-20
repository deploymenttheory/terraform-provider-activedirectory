// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies isolated callbacks at the public read-only membership SDK boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Queries client.GroupMembershipQueries
}

// GroupMembershipQueryService exposes only the explicitly supplied query callbacks.
func (s *Session) GroupMembershipQueryService() client.GroupMembershipQueries { return s.Queries }

// Service refuses omitted callbacks instead of inventing a complete empty result or contacting a real directory.
type Service struct {
	Resolve func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Query   func(context.Context, shared.Identity, memberships.GroupMembersOptions) (memberships.GroupMembersResult, error)
}

// ResolveIdentity forwards the explicit callback with the requested projection.
func (s *Service) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	if s.Resolve == nil {
		return shared.IdentityResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Resolve(ctx, target, options)
}

// GroupMembers forwards direct/recursive mode and all five aggregate budgets without substituting defaults.
func (s *Service) GroupMembers(ctx context.Context, target shared.Identity, options memberships.GroupMembersOptions) (memberships.GroupMembersResult, error) {
	if s.Query == nil {
		return memberships.GroupMembersResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Query(ctx, target, options)
}
