// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

var ErrObservation = errors.New("account settings observation cannot establish the selected identity")

// Snapshot retains SDK observations separately from a usable combined state and positive absence proof.
type Snapshot struct {
	Control         accounts.AccountControlResult
	Expiration      accounts.AccountExpirationResult
	Values          map[string]attr.Value
	Diagnostics     diag.Diagnostics
	ConfirmedAbsent bool
}

// Read resolves the account once, pins the expiration read to that GUID, and performs no mutation or automatic retry.
func Read(ctx context.Context, service client.AccountSettings, target shared.Identity) (Snapshot, error) {
	var result Snapshot
	if service == nil {
		return result, ErrObservation
	}
	var err error
	result.Control, err = service.AccountControl(ctx, target, shared.ResolveIdentityOptions{Properties: []string{"distinguishedName"}})
	if err != nil {
		result.ConfirmedAbsent = lifecycle.ConfirmedAbsent(result.Control.Observation, err)
		return result, fmt.Errorf("read account settings: %w", err)
	}
	if result.Control.Account == nil {
		return result, ErrObservation
	}
	value, err := result.Control.Account.Raw.Values("objectGUID").Single()
	if err != nil {
		return result, ErrObservation
	}
	guid, err := value.GUID()
	if err != nil || guid == (shared.GUID{}) || (target.Form() == shared.IdentityFormGUID && guid.String() != target.String()) {
		return result, ErrObservation
	}
	pinned, err := shared.IdentityByGUID(guid.String())
	if err != nil {
		return result, ErrObservation
	}
	result.Expiration, err = service.AccountExpiration(ctx, pinned, shared.ResolveIdentityOptions{Properties: []string{"distinguishedName"}})
	if err != nil {
		return result, fmt.Errorf("read account settings: %w", err)
	}
	if result.Expiration.Expiration == nil {
		return result, ErrObservation
	}
	result.Values, result.Diagnostics = Project(*result.Control.Account, *result.Expiration.Expiration)
	if result.Diagnostics.HasError() {
		return result, ErrObservation
	}
	return result, nil
}
