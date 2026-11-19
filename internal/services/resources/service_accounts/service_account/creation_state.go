// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// observeCreationStages retains acknowledged same-GUID stages without retrieving or probing any password.
func observeCreationStages(model *ServiceAccountResourceModel, version int64, kind service_accounts.ServiceAccountKind, enableRequested bool,
	result service_accounts.ServiceAccountCreationResult, err error,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	model.PasswordWOVersion = types.Int64Value(0)
	if kind == service_accounts.ServiceAccountStandalone {
		passwordAccepted := result.Password != nil && result.Password.State == shared.ObjectWriteAccepted && result.Password.TargetGUID != nil &&
			result.Password.TargetGUID.String() == model.ID.ValueString()
		if passwordAccepted {
			model.PasswordWOVersion = types.Int64Value(version)
		} else {
			diagnostics.AddError(
				"Service account password stage unacknowledged",
				"The SDK did not acknowledge sMSA password setup for the retained GUID. Reconcile before another write.",
			)
		}
		if enableRequested && (result.Enable == nil || result.Enable.State != shared.ObjectWriteAccepted || !result.Enable.Verified ||
			result.Enable.TargetGUID == nil || result.Enable.TargetGUID.String() != model.ID.ValueString()) {
			diagnostics.AddError("Service account enable stage incomplete", "Requested sMSA enablement was not verified. Earlier accepted stages and GUID are retained.")
		}
	}
	if result.Account != nil && !model.ID.IsNull() && !model.ID.IsUnknown() {
		diagnostics.Append(mapState(model, result.Account)...)
	}
	if err == nil && !result.Complete {
		diagnostics.AddError("Service account creation stages incomplete", "The SDK did not complete every creation stage. Retain identity and reconcile before another write.")
	}
	return diagnostics
}
