// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package mocks supplies SDK-boundary fixtures for isolated provider and feature tests.
package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

var ErrUnexpectedCall = errors.New("unexpected SDK mock call")

// Session provides explicit SDK callbacks. Configure callbacks before using a session concurrently.
// An omitted callback fails rather than contacting a live directory or fabricating an observation.
type Session struct {
	RootDSEFunc    func(context.Context, shared.RootDSEOptions) (shared.RootDSE, error)
	ProtectionFunc func(context.Context) (activedirectory.SessionProtection, error)
	CloseFunc      func() error
	mu             sync.Mutex
	reads          int
	observations   int
	closed         bool
	closeOnce      sync.Once
	closeErr       error
}

// Factory creates a fresh mock session for every provider Configure call.
// It exercises real configuration and protection validation while replacing only SDK I/O.
func Factory(newSession func() client.Session) client.Factory {
	return func(ctx context.Context, options config.Config) (*client.Bundle, error) {
		bundle, err := client.OpenWithDial(ctx, options, func(context.Context, config.Config) (client.Session, error) {
			return newSession(), nil
		})
		if err != nil {
			return nil, fmt.Errorf("configure SDK mock: %w", err)
		}
		return bundle, nil
	}
}

// RootDSE invokes only the configured read callback and records the call.
func (s *Session) RootDSE(ctx context.Context, options shared.RootDSEOptions) (shared.RootDSE, error) {
	s.mu.Lock()
	closed := s.closed
	s.reads++
	s.mu.Unlock()
	if closed {
		return shared.RootDSE{}, shared.ErrClosed
	}
	if s.RootDSEFunc == nil {
		return shared.RootDSE{}, ErrUnexpectedCall
	}
	root, err := s.RootDSEFunc(ctx, options)
	if err != nil {
		return root, fmt.Errorf("mock RootDSE read: %w", err)
	}
	return root, nil
}

// Protection invokes only the configured observation callback.
func (s *Session) Protection(ctx context.Context) (activedirectory.SessionProtection, error) {
	s.mu.Lock()
	closed := s.closed
	s.observations++
	s.mu.Unlock()
	if closed {
		return activedirectory.SessionProtection{}, shared.ErrClosed
	}
	if s.ProtectionFunc == nil {
		return activedirectory.SessionProtection{}, ErrUnexpectedCall
	}
	observed, err := s.ProtectionFunc(ctx)
	if err != nil {
		return observed, fmt.Errorf("mock protection observation: %w", err)
	}
	return observed, nil
}

// Close retires the fixture once, including when multiple Framework cleanup paths run.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		if s.CloseFunc != nil {
			if err := s.CloseFunc(); err != nil {
				s.closeErr = fmt.Errorf("close SDK mock: %w", err)
			}
		}
	})
	return s.closeErr
}

// Calls reports observed SDK calls and session retirement without exposing credentials.
func (s *Session) Calls() (reads, observations int, closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads, s.observations, s.closed
}
