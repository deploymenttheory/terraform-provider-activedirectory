// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package acceptance

import (
	"context"
	"errors"
	"time"
)

var ErrCleanupUnverified = errors.New("owned fixture cleanup was not independently verified")

// CleanupReporter is the test reporting boundary used to preserve cleanup failures.
type CleanupReporter interface {
	Helper()
	Cleanup(func())
	Errorf(string, ...any)
}

// RegisterCleanup runs the feature's owned cleanup and independent absence check even after a test fails.
// False/partial results remain failures. Raw errors are withheld because SDK peer text may contain secrets.
func RegisterCleanup(t CleanupReporter, label string, verify func(context.Context) (bool, error)) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := VerifyCleanup(ctx, verify); err != nil {
			t.Errorf("%s: owned fixture cleanup was not independently verified", label)
		}
	})
}

// VerifyCleanup preserves an explicit failure when cleanup returns an error or cannot prove absence.
func VerifyCleanup(ctx context.Context, verify func(context.Context) (bool, error)) error {
	if verify == nil {
		return ErrCleanupUnverified
	}
	verified, err := verify(ctx)
	if err != nil || !verified || ctx.Err() != nil {
		return ErrCleanupUnverified
	}
	return nil
}
