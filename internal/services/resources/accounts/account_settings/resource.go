// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_account_settings"

var (
	_ resource.Resource                   = &AccountSettingsResource{}
	_ resource.ResourceWithConfigure      = &AccountSettingsResource{}
	_ resource.ResourceWithIdentity       = &AccountSettingsResource{}
	_ resource.ResourceWithImportState    = &AccountSettingsResource{}
	_ resource.ResourceWithModifyPlan     = &AccountSettingsResource{}
	_ resource.ResourceWithValidateConfig = &AccountSettingsResource{}
)

// AccountSettingsResource adopts one stable account GUID and never creates or deletes a directory object.
type AccountSettingsResource struct{ client client.AccountSettings }

// NewAccountSettingsResource creates an unconfigured adopted-account resource without opening a connection.
func NewAccountSettingsResource() resource.Resource { return &AccountSettingsResource{} }

// Metadata identifies the public adopted-account settings resource.
func (*AccountSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema declares explicit ownership and separates server-computed status from writable settings.
func (*AccountSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = accountSettingsSchema()
}

// Configure binds the provider's existing qualified SDK session and drops any previous binding first.
func (r *AccountSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Account settings client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.AccountSettings()
	if err != nil {
		resp.Diagnostics.AddError("Account settings client unavailable", "The qualified SDK session did not expose account settings operations.")
	}
}

// IdentitySchema uses the existing nonzero account GUID as the stable import identity.
func (*AccountSettingsResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable existing account GUID."}},
	}
}

// ImportState establishes management of an existing GUID without applying or resetting account settings.
func (*AccountSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	identity.ImportGUID(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	canonical, _ := identity.CanonicalGUID(req.ID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_guid"), canonical)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), canonical)...)
	}
}

// ValidateConfig rejects known invalid multi-phase intent before provider configuration or SDK access.
func (*AccountSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AccountSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.AccountGUID.IsUnknown() || config.ExpiresAt.IsUnknown() || config.NeverExpires.IsUnknown() {
		return
	}
	patch := accounts.AccountControlPatch{}
	for _, field := range flagProperties(&config, &patch) {
		if field.value.IsUnknown() {
			return
		}
	}
	_, diagnostics := constructSettings(&config)
	resp.Diagnostics.Append(diagnostics...)
}

// ModifyPlan invalidates the old observed GUID when management is replaced with a different existing target.
func (*AccountSettingsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var planned, previous AccountSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	if resp.Diagnostics.HasError() {
		return
	}
	next, nextErr := identity.CanonicalGUID(planned.AccountGUID.ValueString())
	prior, priorErr := identity.CanonicalGUID(previous.AccountGUID.ValueString())
	if nextErr == nil && priorErr == nil && next != prior {
		planned.ID = types.StringUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &planned)...)
	}
}
