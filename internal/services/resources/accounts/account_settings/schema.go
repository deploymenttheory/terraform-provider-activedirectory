// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
)

// accountSettingsSchema distinguishes configured ownership from observed state on an existing account.
func accountSettingsSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "Stable existing account object GUID and import identity.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"account_guid": schema.StringAttribute{
			Required:      true,
			Description:   "Explicit existing account object GUID. Changing the target relinquishes the previous account and adopts the new one; neither account is created or deleted.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current observed distinguished name; external rename/move retains GUID identity."},
		"expires_at": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Finite canonical UTC RFC3339 expiration at 100ns precision or coarser. Omission owns no expiration. Set never_expires = true to clear explicitly.",
		},
		"never_expires": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Description: "Explicit true clears expiration through the SDK. False requires expires_at. Omission owns no expiration; both " +
				"native never-expire representations are observed as true.",
		},
		"account_expires": schema.Int64Attribute{
			Computed: true,
			Description: "Observed raw nonnegative FILETIME expiration. Zero and 9223372036854775807 are distinct native never-expire " +
				"representations; this output is never written directly.",
		},
		"password_last_set": schema.Int64Attribute{Computed: true, Description: "Observed pwdLastSet FILETIME when visible; unavailable is null."},
		"locked_out": schema.BoolAttribute{
			Computed:    true,
			Description: "Selected DC's computed lockout state when visible. This resource never writes a computed flag or unlocks an account.",
		},
		"password_expired": schema.BoolAttribute{Computed: true, Description: "Selected DC's computed password-expiry state when visible. This value is never written."},
		"change_password_at_logon": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Description: "Explicit pwdLastSet intent: true requires change at next logon; false asks the server to set its current " +
				"password timestamp. Already-matching intent is not replayed. Omission leaves it unmanaged.",
		},
	}
	model, patch := AccountSettingsResourceModel{}, accounts.AccountControlPatch{}
	for _, field := range flagProperties(&model, &patch) {
		if field.name == "change_password_at_logon" {
			continue
		}
		attributes[field.name] = schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Explicit typed " + field.name + " account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.",
		}
	}
	return schema.Schema{
		Description: "Adopt an existing account and manage only explicitly configured account-control/expiration settings. Destroy " +
			"relinquishes ownership without deleting or resetting the account. " +
			"Do not co-own the same fields in typed account resources or other Terraform states; cross-workspace conflicts are not detected.",
		Attributes: attributes,
	}
}
