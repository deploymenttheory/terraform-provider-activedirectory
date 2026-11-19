// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Computers binds typed computer lifecycle, generic location changes and explicit password resets to one SDK session.
type Computers interface {
	Computer(context.Context, shared.Identity, shared.ResolveIdentityOptions) (computers.ComputerResult, error)
	Computers(context.Context, shared.SearchOptions) (computers.ComputersResult, error)
	CreateComputer(context.Context, computers.CreateComputerRequest) (computers.ComputerCreationResult, error)
	UpdateComputer(context.Context, computers.UpdateComputerRequest) (computers.ComputerMutationResult, error)
	DeleteComputer(context.Context, computers.DeleteComputerRequest) (computers.ComputerMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
	ResetAccountPassword(context.Context, accounts.PasswordResetRequest) (accounts.PasswordResult, error)
}

var ErrComputers = errors.New("configured SDK session does not expose computer operations")

// Computers returns services sharing the bundle's authenticated connection, protection and lifetime.
func (b *Bundle) Computers() (Computers, error) {
	if b == nil || b.session == nil {
		return nil, ErrComputers
	}
	getter, ok := b.session.(interface{ ComputerService() Computers })
	if !ok {
		return nil, ErrComputers
	}
	service := getter.ComputerService()
	if service == nil {
		return nil, ErrComputers
	}
	return service, nil
}

// ComputerService combines public SDK services without opening another connection or changing authentication.
func (s *sdkSession) ComputerService() Computers {
	return &sdkComputers{Service: s.Computers, session: s}
}

type sdkComputers struct {
	*computers.Service
	session *sdkSession
}

// RenameObject changes the same GUID's name using the generic SDK operation.
func (s *sdkComputers) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	result, err := s.session.Directory.RenameObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("rename computer: %w", err)
	}
	return result, nil
}

// MoveObject delegates explicit same-domain location changes on the shared session.
func (s *sdkComputers) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	result, err := s.session.Directory.MoveObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("move computer: %w", err)
	}
	return result, nil
}

// ResetAccountPassword submits one password operation; the caller owns and clears the UTF-8 buffer.
func (s *sdkComputers) ResetAccountPassword(ctx context.Context, request accounts.PasswordResetRequest) (accounts.PasswordResult, error) {
	result, err := s.session.Accounts.ResetAccountPassword(ctx, request)
	if err != nil {
		return result, fmt.Errorf("reset computer password: %w", err)
	}
	return result, nil
}
