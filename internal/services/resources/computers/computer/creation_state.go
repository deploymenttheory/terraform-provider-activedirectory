// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// observeCreationStages retains acknowledged same-GUID stages and diagnoses incomplete creation without adopting a typed candidate identity.
func observeCreationStages(model *ComputerResourceModel, version int64, passwordRequested, enableRequested bool,
	result computers.ComputerCreationResult, err error,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	model.PasswordWOVersion = types.Int64Value(0)
	passwordAccepted := result.Password != nil && result.Password.State == shared.ObjectWriteAccepted && result.Password.TargetGUID != nil &&
		result.Password.TargetGUID.String() == model.ID.ValueString()
	if passwordAccepted {
		model.PasswordWOVersion = types.Int64Value(version)
	}
	if result.Computer != nil && !model.ID.IsNull() && !model.ID.IsUnknown() {
		diagnostics.Append(mapState(model, result.Computer)...)
	}
	if passwordRequested && !passwordAccepted {
		diagnostics.AddError("Computer password stage unacknowledged",
			"The SDK did not acknowledge password setup for this GUID. The accepted creation and unresolved intent are retained.")
	}
	if enableRequested && (result.Enable == nil || result.Enable.State != shared.ObjectWriteAccepted || !result.Enable.Verified ||
		result.Enable.TargetGUID == nil || result.Enable.TargetGUID.String() != model.ID.ValueString()) {
		diagnostics.AddError("Computer enable stage incomplete", "The SDK did not verify requested enablement. Earlier accepted stages and GUID are retained.")
	}
	if err == nil && !result.Complete {
		diagnostics.AddError(
			"Computer creation stages incomplete",
			"The SDK did not complete every requested stage. Preserve accepted identity and reconcile before another write.",
		)
	}
	return diagnostics
}
