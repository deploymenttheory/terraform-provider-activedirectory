// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func TestLaterEnableUncertaintyPreservesAcknowledgedPasswordStage(t *testing.T) {
	model := creationModel()
	model.ID = types.StringValue(fixtureGUID)
	guid, _ := shared.ParseGUID(fixtureGUID)
	result := computers.ComputerCreationResult{
		Password: &accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid},
		Enable:   &shared.ObjectMutationResult{State: shared.ObjectWriteUncertain, TargetGUID: &guid},
	}
	if diagnostics := observeCreationStages(&model, 3, true, true, result, shared.ErrObjectWriteUncertain); !diagnostics.HasError() || model.PasswordWOVersion.ValueInt64() != 3 {
		t.Fatal("uncertain later enablement discarded an acknowledged password stage")
	}
	result.Password.State = shared.ObjectWriteUncertain
	if diagnostics := observeCreationStages(&model, 3, true, true, result, accounts.ErrPasswordUncertain); !diagnostics.HasError() || model.PasswordWOVersion.ValueInt64() != 0 {
		t.Fatal("an uncertain password stage was recorded as acknowledged")
	}
}

func TestTypedCreationCandidateCannotSupplyUnobservedGenericGUID(t *testing.T) {
	directory := newUnitDirectory(t)
	model := creationModel()
	result := computers.ComputerCreationResult{Computer: directory.snapshot(t)}
	observeCreationStages(&model, 0, false, false, result, shared.ErrObjectWriteUncertain)
	if !model.ID.IsNull() {
		t.Fatal("an unverified typed candidate became the creation identity")
	}
}
