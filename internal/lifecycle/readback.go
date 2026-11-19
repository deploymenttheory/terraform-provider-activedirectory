// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package lifecycle

import (
	"context"
	"errors"
	"time"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var ErrReadback = errors.New("bounded directory readback did not establish the required observation")

// ReadbackPolicy bounds read-only reconciliation. It cannot contain a mutation callback.
type ReadbackPolicy struct {
	Attempts int
	Delay    time.Duration
	Timeout  time.Duration
}

// ReadObservation preserves each attempt's partial SDK result and error for feature-specific reconciliation.
// Errors are internal evidence only; never place their raw text in Terraform diagnostics or logs.
type ReadObservation struct {
	Result shared.IdentityResult
	Err    error
}

// Readback performs at most five reads inside a bounded context, retaining every returned observation.
// Only incomplete/not-found observations are eligible for another read. Denials and unknown failures stop immediately.
// Callers inspect the returned SDK result as well as err; this helper never reapplies a write.
func Readback(
	ctx context.Context,
	policy ReadbackPolicy,
	read func(context.Context) (shared.IdentityResult, error),
	ready func(shared.IdentityResult) bool,
) ([]ReadObservation, error) {
	if policy.Attempts < 1 || policy.Attempts > 5 || policy.Delay < 0 || policy.Delay > 2*time.Second ||
		policy.Timeout <= 0 || policy.Timeout > 30*time.Second || read == nil || ready == nil {
		return nil, ErrReadback
	}
	ctx, cancel := context.WithTimeout(ctx, policy.Timeout)
	defer cancel()
	var observations []ReadObservation
	for attempt := 0; attempt < policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return observations, errors.Join(ErrReadback, err)
		}
		result, err := read(ctx)
		observations = append(observations, ReadObservation{Result: result, Err: err})
		if cause := ctx.Err(); cause != nil {
			return observations, errors.Join(ErrReadback, err, cause)
		}
		if err == nil && result.Complete && result.Object != nil && ready(result) {
			return observations, nil
		}
		incomplete := shared.Classify(err) == shared.ClassIncomplete && onlyReadErrors(err, true)
		if err != nil && !incomplete && !ConfirmedAbsent(result, err) {
			return observations, errors.Join(ErrReadback, err)
		}
		if attempt+1 < policy.Attempts {
			timer := time.NewTimer(policy.Delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return observations, errors.Join(ErrReadback, ctx.Err())
			case <-timer.C:
			}
		}
	}
	return observations, ErrReadback
}
