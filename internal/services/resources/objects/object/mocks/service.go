// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies explicit callbacks at the public SDK generic object boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Objects client.Objects
}

// ObjectService exposes only the supplied generic object callbacks on this synthetic authenticated session.
func (s *Session) ObjectService() client.Objects { return s.Objects }

// Service rejects every omitted callback rather than inventing successful writes or contacting AD.
type Service struct {
	Read   func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	List   func(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	Create func(context.Context, shared.CreateObjectRequest) (shared.ObjectMutationResult, error)
	Update func(context.Context, shared.UpdateObjectRequest) (shared.ObjectMutationResult, error)
	Delete func(context.Context, shared.DeleteObjectRequest) (shared.ObjectMutationResult, error)
	Rename func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move   func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

// ResolveIdentity forwards an explicit generic GUID or DN lookup.
func (s *Service) ResolveIdentity(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
	if s.Read == nil {
		return shared.IdentityResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, target, options)
}

// Search forwards an explicit bounded generic search.
func (s *Service) Search(ctx context.Context, options shared.SearchOptions) (shared.SearchResult, error) {
	if s.List == nil {
		return shared.SearchResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateObject forwards a single object creation request.
func (s *Service) CreateObject(ctx context.Context, request shared.CreateObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Create == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateObject forwards only explicitly constructed changes.
func (s *Service) UpdateObject(ctx context.Context, request shared.UpdateObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Update == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteObject forwards a leaf-only object deletion request.
func (s *Service) DeleteObject(ctx context.Context, request shared.DeleteObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Delete == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
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
