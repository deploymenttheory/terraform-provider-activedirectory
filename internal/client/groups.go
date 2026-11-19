// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// Groups binds group lifecycle and location changes to the provider's existing qualified SDK session.
type Groups interface {
	Group(context.Context, shared.Identity, shared.ResolveIdentityOptions) (groups.GroupResult, error)
	Groups(context.Context, shared.SearchOptions) (groups.GroupsResult, error)
	CreateGroup(context.Context, groups.CreateGroupRequest) (groups.GroupMutationResult, error)
	UpdateGroup(context.Context, groups.UpdateGroupRequest) (groups.GroupMutationResult, error)
	DeleteGroup(context.Context, groups.DeleteGroupRequest) (groups.GroupMutationResult, error)
	RenameObject(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error)
	MoveObject(context.Context, shared.MoveObjectRequest) (shared.ObjectMoveResult, error)
}

var ErrGroups = errors.New("configured SDK session does not expose group operations")

// Groups returns services with the same authenticated connection and lifetime as provider configuration.
func (b *Bundle) Groups() (Groups, error) {
	if b == nil || b.session == nil {
		return nil, ErrGroups
	}
	getter, ok := b.session.(interface{ GroupService() Groups })
	if !ok {
		return nil, ErrGroups
	}
	service := getter.GroupService()
	if service == nil {
		return nil, ErrGroups
	}
	return service, nil
}

// GroupService composes public SDK operations without opening another session or managing membership.
func (s *sdkSession) GroupService() Groups { return &sdkGroups{Service: s.Groups, session: s} }

type sdkGroups struct {
	*groups.Service
	session *sdkSession
}

// RenameObject changes the same group's name through the generic SDK operation.
func (s *sdkGroups) RenameObject(ctx context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
	result, err := s.session.Directory.RenameObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("rename group: %w", err)
	}
	return result, nil
}

// MoveObject changes the group's explicit location without changing its GUID or group membership.
func (s *sdkGroups) MoveObject(ctx context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
	result, err := s.session.Directory.MoveObject(ctx, request)
	if err != nil {
		return result, fmt.Errorf("move group: %w", err)
	}
	return result, nil
}
