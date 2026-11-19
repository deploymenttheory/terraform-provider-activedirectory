// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ActionName = "activedirectory_account_unlock"

var (
	_ action.Action                   = &AccountUnlockAction{}
	_ action.ActionWithConfigure      = &AccountUnlockAction{}
	_ action.ActionWithValidateConfig = &AccountUnlockAction{}
)

// AccountUnlockAction resets lockoutTime for one account without changing its enabled status.
type AccountUnlockAction struct {
	client       client.Accounts
	capabilities shared.DirectoryCapabilities
	server       string
}

// NewAccountUnlockAction constructs an action without opening a directory connection.
func NewAccountUnlockAction() action.Action { return &AccountUnlockAction{} }

// Metadata reports the public Terraform action type.
func (a *AccountUnlockAction) Metadata(_ context.Context, _ action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = ActionName
}

// Schema requires a stable GUID and the exact observed DC name to make the operation's scope explicit.
func (a *AccountUnlockAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Unlock one AD DS account on the selected DC by resetting lockoutTime to zero. " +
			"Does not enable a disabled account or prove forest-wide convergence. Failed or uncertain submissions are never automatically retried.",
		Attributes: map[string]schema.Attribute{
			"object_guid": schema.StringAttribute{Required: true, MarkdownDescription: "Nonzero GUID of the account to unlock. Names and DNs are not accepted as stable identity."},
			"server": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact DC DNS name observed by the configured provider. Must match RootDSE dnsHostName; no server discovery or failover is performed.",
			},
		},
	}
}

// Configure binds the action to the shared SDK session after checking directory type and observed server identity.
func (a *AccountUnlockAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = nil
	a.server = ""
	a.capabilities = shared.DirectoryCapabilities{}
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Account client unavailable", "The action requires a configured Active Directory client bundle.")
		return
	}
	capabilities := bundle.Capabilities()
	if capabilities.RequireADDS(ActionName) != nil {
		resp.Diagnostics.AddError("Account unlock unsupported", "The selected server must advertise AD DS. Capability observations do not establish write permission.")
		return
	}
	server, known := bundle.ServerName()
	if !known {
		resp.Diagnostics.AddError("Account unlock server unobserved", "The server did not return its DNS identity. Select and verify the intended DC before invoking this action.")
		return
	}
	service, err := bundle.Accounts()
	if err != nil {
		resp.Diagnostics.AddError("Account client unavailable", "The configured SDK session did not expose account operations.")
		return
	}
	a.client, a.capabilities, a.server = service, capabilities, server
}
