// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// OrganizationalUnits is the public SDK boundary used by OU lifecycle and discovery services.
type OrganizationalUnits interface {
	OrganizationalUnit(context.Context, shared.Identity, shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error)
	OrganizationalUnits(context.Context, shared.SearchOptions) (organizational_units.OrganizationalUnitsResult, error)
	CreateOrganizationalUnit(context.Context, organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error)
	UpdateOrganizationalUnit(context.Context, organizational_units.UpdateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error)
	DeleteOrganizationalUnit(context.Context, organizational_units.DeleteOrganizationalUnitRequest) (shared.ObjectMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

var ErrOrganizationalUnits = errors.New("configured SDK session does not expose organizational units")

// OrganizationalUnits returns OU services sharing this bundle's authenticated session and lifetime.
func (b *Bundle) OrganizationalUnits() (OrganizationalUnits, error) {
	if b == nil || b.session == nil {
		return nil, ErrOrganizationalUnits
	}
	bound, ok := b.session.(interface{ OrganizationalUnitService() OrganizationalUnits })
	if !ok {
		return nil, ErrOrganizationalUnits
	}
	service := bound.OrganizationalUnitService()
	if service == nil {
		return nil, ErrOrganizationalUnits
	}
	return service, nil
}

func (s *sdkSession) OrganizationalUnitService() OrganizationalUnits {
	return &sdkOrganizationalUnits{Service: s.OrganizationalUnits, session: s}
}

type sdkOrganizationalUnits struct {
	*organizational_units.Service
	session *sdkSession
}

// RenameObject delegates to the generic SDK operation on the same authenticated session.
func (s *sdkOrganizationalUnits) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	result, err := s.session.Directory.RenameObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("rename organizational unit: %w", err)
	}
	return result, nil
}

// MoveObject delegates a same-domain move; feature construction leaves Destination unset.
func (s *sdkOrganizationalUnits) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	result, err := s.session.Directory.MoveObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("move organizational unit: %w", err)
	}
	return result, nil
}
