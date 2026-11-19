// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/mocks"
)

func TestSessionFailsUnexpectedCallsAndRetiresOnce(t *testing.T) {
	t.Parallel()
	closed := 0
	session := &mocks.Session{CloseFunc: func() error { closed++; return nil }}
	if _, err := session.RootDSE(context.Background(), shared.RootDSEOptions{}); !errors.Is(err, mocks.ErrUnexpectedCall) {
		t.Fatal("unexpected read was accepted")
	}
	if _, err := session.Protection(context.Background()); !errors.Is(err, mocks.ErrUnexpectedCall) {
		t.Fatal("missing observation was fabricated")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatal("close callback ran more than once")
	}
	if _, err := session.RootDSE(context.Background(), shared.RootDSEOptions{}); !errors.Is(err, shared.ErrClosed) {
		t.Fatal("closed session remained usable")
	}
}
