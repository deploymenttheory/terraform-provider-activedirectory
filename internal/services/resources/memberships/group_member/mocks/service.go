// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies isolated direct-link callbacks at the public SDK boundary.
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
	Memberships client.Memberships
}

// MembershipService exposes the explicitly supplied direct-link operations.
func (s *Session) MembershipService() client.Memberships { return s.Memberships }

// Service refuses omitted callbacks instead of contacting a directory or inventing a successful mutation.
type Service struct {
	Resolve func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Assert  func(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	Links   func(context.Context, shared.Identity, memberships.GroupMemberLinksOptions) (memberships.GroupMemberLinksResult, error)
	Add     func(context.Context, memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error)
	Remove  func(context.Context, memberships.RemoveGroupMembersRequest) (memberships.GroupMembershipMutationResult, error)
}

// ResolveIdentity forwards only the explicitly configured callback.
func (s *Service) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	if s.Resolve == nil {
		return shared.IdentityResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Resolve(ctx, target, options)
}

// Search forwards only the explicitly configured callback.
func (s *Service) Search(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	if s.Assert == nil {
		return shared.SearchResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Assert(ctx, options)
}

// GroupMemberLinks forwards only the explicitly configured callback.
func (s *Service) GroupMemberLinks(ctx context.Context, target shared.Identity, options memberships.GroupMemberLinksOptions) (memberships.GroupMemberLinksResult, error) {
	if s.Links == nil {
		return memberships.GroupMemberLinksResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Links(ctx, target, options)
}

// AddGroupMembers forwards only the explicitly configured callback.
func (s *Service) AddGroupMembers(ctx context.Context, request memberships.AddGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
	if s.Add == nil {
		return memberships.GroupMembershipMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Add(ctx, request)
}

// RemoveGroupMembers forwards only the explicitly configured callback.
func (s *Service) RemoveGroupMembers(ctx context.Context, request memberships.RemoveGroupMembersRequest) (memberships.GroupMembershipMutationResult, error) {
	if s.Remove == nil {
		return memberships.GroupMembershipMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Remove(ctx, request)
}
