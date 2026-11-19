// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Users binds typed user lifecycle, generic location changes and explicit password resets to one SDK session.
type Users interface {
	User(context.Context, shared.Identity, shared.ResolveIdentityOptions) (users.UserResult, error)
	Users(context.Context, shared.SearchOptions) (users.UsersResult, error)
	CreateUser(context.Context, users.CreateUserRequest) (users.UserCreationResult, error)
	UpdateUser(context.Context, users.UpdateUserRequest) (users.UserMutationResult, error)
	DeleteUser(context.Context, users.DeleteUserRequest) (users.UserMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
	ResetAccountPassword(context.Context, accounts.PasswordResetRequest) (accounts.PasswordResult, error)
}

var ErrUsers = errors.New("configured SDK session does not expose user operations")

// Users returns services sharing the bundle's authenticated connection, protection and lifetime.
func (b *Bundle) Users() (Users, error) {
	if b == nil || b.session == nil {
		return nil, ErrUsers
	}
	getter, ok := b.session.(interface{ UserService() Users })
	if !ok {
		return nil, ErrUsers
	}
	service := getter.UserService()
	if service == nil {
		return nil, ErrUsers
	}
	return service, nil
}

// UserService combines public SDK services without opening another connection or changing authentication.
func (s *sdkSession) UserService() Users { return &sdkUsers{Service: s.Users, session: s} }

type sdkUsers struct {
	*users.Service
	session *sdkSession
}

// RenameObject changes the same GUID's name using the generic SDK operation.
func (s *sdkUsers) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	result, err := s.session.Directory.RenameObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("rename user: %w", err)
	}
	return result, nil
}

// MoveObject delegates explicit same-domain location changes on the shared session.
func (s *sdkUsers) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	result, err := s.session.Directory.MoveObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("move user: %w", err)
	}
	return result, nil
}

// ResetAccountPassword submits one password operation; the caller owns and clears the UTF-8 buffer.
func (s *sdkUsers) ResetAccountPassword(ctx context.Context, request accounts.PasswordResetRequest) (accounts.PasswordResult, error) {
	result, err := s.session.Accounts.ResetAccountPassword(ctx, request)
	if err != nil {
		return result, fmt.Errorf("reset user password: %w", err)
	}
	return result, nil
}
