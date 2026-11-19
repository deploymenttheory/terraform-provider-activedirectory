// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies explicit callbacks at the public SDK computer boundary.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

type Session struct {
	*sdkmocks.Session
	Computers client.Computers
}

// ComputerService exposes only the supplied computer callbacks on this synthetic authenticated session.
func (s *Session) ComputerService() client.Computers { return s.Computers }

// Service rejects every omitted callback rather than inventing successful writes or contacting AD.
type Service struct {
	Read     func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (computers.ComputerResult, error)
	List     func(context.Context, shared.SearchOptions) (computers.ComputersResult, error)
	Create   func(context.Context, computers.CreateComputerRequest) (computers.ComputerCreationResult, error)
	Update   func(context.Context, computers.UpdateComputerRequest) (computers.ComputerMutationResult, error)
	Delete   func(context.Context, computers.DeleteComputerRequest) (computers.ComputerMutationResult, error)
	Rename   func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move     func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
	Password func(context.Context, accounts.PasswordResetRequest) (accounts.PasswordResult, error)
}

// Computer forwards an explicit lookup.
func (s *Service) Computer(ctx context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (computers.ComputerResult, error) {
	if s.Read == nil {
		return computers.ComputerResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, target, options)
}

// Computers forwards an explicit bounded search.
func (s *Service) Computers(ctx context.Context, options shared.SearchOptions) (computers.ComputersResult, error) {
	if s.List == nil {
		return computers.ComputersResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateComputer forwards a single staged creation request.
func (s *Service) CreateComputer(ctx context.Context, request computers.CreateComputerRequest) (computers.ComputerCreationResult, error) {
	if s.Create == nil {
		return computers.ComputerCreationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateComputer forwards only explicitly constructed changes.
func (s *Service) UpdateComputer(ctx context.Context, request computers.UpdateComputerRequest) (computers.ComputerMutationResult, error) {
	if s.Update == nil {
		return computers.ComputerMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteComputer forwards a leaf-only computer deletion request.
func (s *Service) DeleteComputer(ctx context.Context, request computers.DeleteComputerRequest) (computers.ComputerMutationResult, error) {
	if s.Delete == nil {
		return computers.ComputerMutationResult{}, sdkmocks.ErrUnexpectedCall
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
