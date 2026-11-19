// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"

	sdkclient "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/client"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Objects exposes generic object operations on the provider's existing qualified directory session.
// Mutation results retain accepted, rejected and uncertain outcomes alongside partial observations.
type Objects interface {
	ResolveIdentity(context.Context, shared.Identity, shared.ResolveIdentityOptions) (shared.IdentityResult, error)
	Search(context.Context, shared.SearchOptions) (shared.SearchResult, error)
	CreateObject(context.Context, shared.CreateObjectRequest) (shared.ObjectMutationResult, error)
	UpdateObject(context.Context, shared.UpdateObjectRequest) (shared.ObjectMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
	DeleteObject(context.Context, shared.DeleteObjectRequest) (shared.ObjectMutationResult, error)
}

var (
	ErrObjects         = errors.New("configured SDK session does not expose generic object operations")
	_          Objects = (*sdkclient.Client)(nil)
)

// Objects returns generic operations without opening another connection or changing authentication.
func (b *Bundle) Objects() (Objects, error) {
	if b == nil || b.session == nil {
		return nil, ErrObjects
	}
	getter, ok := b.session.(interface{ ObjectService() Objects })
	if !ok {
		return nil, ErrObjects
	}
	service := getter.ObjectService()
	if service == nil {
		return nil, ErrObjects
	}
	return service, nil
}

// ObjectService shares the configured SDK connection and its lifetime with every generic object surface.
func (s *sdkSession) ObjectService() Objects { return s.Directory }
