// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies explicit callbacks at the public SDK user boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Users client.Users
}

// UserService exposes only the supplied user callbacks on this synthetic authenticated session.
func (s *Session) UserService() client.Users { return s.Users }

// Service rejects every omitted callback rather than inventing successful writes or contacting AD.
type Service struct {
	Read     func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (users.UserResult, error)
	List     func(context.Context, shared.SearchOptions) (users.UsersResult, error)
	Create   func(context.Context, users.CreateUserRequest) (users.UserCreationResult, error)
	Update   func(context.Context, users.UpdateUserRequest) (users.UserMutationResult, error)
	Delete   func(context.Context, users.DeleteUserRequest) (users.UserMutationResult, error)
	Rename   func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move     func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
	Password func(context.Context, accounts.PasswordResetRequest) (accounts.PasswordResult, error)
}

// User forwards an explicit lookup.
func (s *Service) User(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (users.UserResult, error) {
	if s.Read == nil {
		return users.UserResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, target, options)
}

// Users forwards an explicit bounded search.
func (s *Service) Users(ctx context.Context, options shared.SearchOptions) (users.UsersResult, error) {
	if s.List == nil {
		return users.UsersResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateUser forwards a single staged creation request.
func (s *Service) CreateUser(ctx context.Context, request users.CreateUserRequest) (users.UserCreationResult, error) {
	if s.Create == nil {
		return users.UserCreationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateUser forwards only explicitly constructed changes.
func (s *Service) UpdateUser(ctx context.Context, request users.UpdateUserRequest) (users.UserMutationResult, error) {
	if s.Update == nil {
		return users.UserMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteUser forwards a leaf-only user deletion request.
func (s *Service) DeleteUser(ctx context.Context, request users.DeleteUserRequest) (users.UserMutationResult, error) {
	if s.Delete == nil {
		return users.UserMutationResult{}, sdkmocks.ErrUnexpectedCall
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

// ResetAccountPassword forwards a single reset without retaining its secret value.
func (s *Service) ResetAccountPassword(ctx context.Context, request accounts.PasswordResetRequest) (accounts.PasswordResult, error) {
	if s.Password == nil {
		return accounts.PasswordResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Password(ctx, request)
}
