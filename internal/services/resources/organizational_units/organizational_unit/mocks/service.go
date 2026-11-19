// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks provides explicit SDK callbacks for organizational unit tests.
package mocks

import (
	"context"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	sdkmocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

// Service implements only explicitly configured callbacks and never contacts a directory.
type Service struct {
	Read   func(context.Context, shared.Identity, shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error)
	List   func(context.Context, shared.SearchOptions) (organizational_units.OrganizationalUnitsResult, error)
	Create func(context.Context, organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error)
	Update func(context.Context, organizational_units.UpdateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error)
	Delete func(context.Context, organizational_units.DeleteOrganizationalUnitRequest) (shared.ObjectMutationResult, error)
	Rename func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	Move   func(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

var _ client.OrganizationalUnits = &Service{}

// Session adds feature services to the provider's isolated connection fixture.
type Session struct {
	*sdkmocks.Session
	Units client.OrganizationalUnits
}

// OrganizationalUnitService exposes this fixture's OU callbacks on its existing mock session.
func (s *Session) OrganizationalUnitService() client.OrganizationalUnits { return s.Units }

// OrganizationalUnit invokes the explicitly supplied single-object callback.
func (s *Service) OrganizationalUnit(ctx context.Context, id shared.Identity, options shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
	if s.Read == nil {
		return organizational_units.OrganizationalUnitResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Read(ctx, id, options)
}

// OrganizationalUnits invokes the explicitly supplied bounded discovery callback.
func (s *Service) OrganizationalUnits(ctx context.Context, options shared.SearchOptions) (organizational_units.OrganizationalUnitsResult, error) {
	if s.List == nil {
		return organizational_units.OrganizationalUnitsResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.List(ctx, options)
}

// CreateOrganizationalUnit invokes the explicitly supplied Add workflow callback.
func (s *Service) CreateOrganizationalUnit(
	ctx context.Context,
	request organizational_units.CreateOrganizationalUnitRequest,
) (organizational_units.OrganizationalUnitMutationResult, error) {
	if s.Create == nil {
		return organizational_units.OrganizationalUnitMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Create(ctx, request)
}

// UpdateOrganizationalUnit invokes the explicitly supplied Modify/protection callback.
func (s *Service) UpdateOrganizationalUnit(
	ctx context.Context,
	request organizational_units.UpdateOrganizationalUnitRequest,
) (organizational_units.OrganizationalUnitMutationResult, error) {
	if s.Update == nil {
		return organizational_units.OrganizationalUnitMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Update(ctx, request)
}

// DeleteOrganizationalUnit invokes the explicitly supplied leaf-delete callback.
func (s *Service) DeleteOrganizationalUnit(ctx context.Context, request organizational_units.DeleteOrganizationalUnitRequest) (shared.ObjectMutationResult, error) {
	if s.Delete == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Delete(ctx, request)
}

// RenameObject invokes the explicitly supplied same-GUID rename callback.
func (s *Service) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	if s.Rename == nil {
		return shared.ObjectMutationResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Rename(ctx, request)
}

// MoveObject invokes the explicitly supplied same-domain move callback.
func (s *Service) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	if s.Move == nil {
		return shared.ObjectMoveResult{}, sdkmocks.ErrUnexpectedCall
	}
	return s.Move(ctx, request)
}
