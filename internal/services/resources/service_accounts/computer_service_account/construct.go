// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// constructPair validates the whole operation and refuses disagreement with an already established composite identity.
func constructPair(model ComputerServiceAccountResourceModel) (shared.Identity, shared.Identity, string, diag.Diagnostics) {
	computer, account, diagnostics := common.ConstructPair(model.ComputerGUID, model.AccountGUID)
	if diagnostics.HasError() {
		return computer, account, "", diagnostics
	}
	id, _ := identity.RelationshipID(computer.String(), account.String())
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != id {
		diagnostics.AddError("Conflicting association identity", "The stored composite ID disagrees with the endpoint GUIDs. No directory operation was performed.")
	}
	return computer, account, id, diagnostics
}

// constructAdd submits exactly one persistent direct link with native permissive duplicate-add semantics.
func constructAdd(computer, account shared.Identity) sdkaccounts.ComputerServiceAccountRequest {
	return sdkaccounts.ComputerServiceAccountRequest{Computer: computer, Accounts: []shared.Identity{account}, Permissive: true}
}

// constructRemove removes exactly the owned link with native permissive missing-link semantics.
func constructRemove(computer, account shared.Identity) sdkaccounts.ComputerServiceAccountRequest {
	return sdkaccounts.ComputerServiceAccountRequest{Computer: computer, Accounts: []shared.Identity{account}, Permissive: true}
}
