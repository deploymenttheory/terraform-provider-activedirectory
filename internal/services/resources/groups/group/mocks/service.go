// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies explicit callbacks at the public SDK user boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Groups client.Groups
}

// GroupService exposes only the supplied user callbacks on this synthetic authenticated session.
func (s *Session) GroupService() client.Groups { return s.Groups }

// Service rejects every omitted callback rather than inventing successful writes or contacting AD.
type Service struct {
	Read   func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (groups.GroupResult, error)
	List   func(context.Context, shared.SearchOptions) (groups.GroupsResult, error)
	Create func(context.Context, groups.CreateGroupRequest) (groups.GroupMutationResult, error)
	Update func(context.Context, groups.UpdateGroupRequest) (groups.GroupMutationResult, error)
	Delete func(context.Context, groups.DeleteGroupRequest) (groups.GroupMutationResult, error)
	Rename func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move   func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

// Group forwards an explicit lookup.
func (s *Service) Group(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (groups.GroupResult, error) {
	if s.Read == nil {
		return groups.GroupResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, target, options)
}

// Groups forwards an explicit bounded search.
func (s *Service) Groups(ctx context.Context, options shared.SearchOptions) (groups.GroupsResult, error) {
	if s.List == nil {
		return groups.GroupsResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateGroup forwards a single group creation request.
func (s *Service) CreateGroup(ctx context.Context, request groups.CreateGroupRequest) (groups.GroupMutationResult, error) {
	if s.Create == nil {
		return groups.GroupMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateGroup forwards only explicitly constructed changes.
func (s *Service) UpdateGroup(ctx context.Context, request groups.UpdateGroupRequest) (groups.GroupMutationResult, error) {
	if s.Update == nil {
		return groups.GroupMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteGroup forwards a leaf-only user deletion request.
func (s *Service) DeleteGroup(ctx context.Context, request groups.DeleteGroupRequest) (groups.GroupMutationResult, error) {
	if s.Delete == nil {
		return groups.GroupMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Delete(ctx, request)
}

// RenameObject forwards a same-GUID rename.
func (s *Service) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Rename == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Rename(ctx, request)
}

// MoveObject forwards an explicitly selected same-domain move.
func (s *Service) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	if s.Move == nil {
		return shared.ObjectMoveResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Move(ctx, request)
}
