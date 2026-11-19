// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_service_account"

var (
	_ resource.Resource                = &ServiceAccountResource{}
	_ resource.ResourceWithConfigure   = &ServiceAccountResource{}
	_ resource.ResourceWithIdentity    = &ServiceAccountResource{}
	_ resource.ResourceWithImportState = &ServiceAccountResource{}
	_ resource.ResourceWithModifyPlan  = &ServiceAccountResource{}
)

// ServiceAccountResource manages one service account by GUID, preserving staged effects and never exposing a password in state.
type ServiceAccountResource struct{ client client.ServiceAccounts }

// NewServiceAccountResource creates an unconfigured resource without opening a directory connection.
func NewServiceAccountResource() resource.Resource { return &ServiceAccountResource{} }

// Metadata identifies the public service account resource type.
func (*ServiceAccountResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema defines typed property ownership and versioned write-only password input.
func (*ServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = serviceAccountSchema()
}

// Configure binds the provider's qualified SDK session and retires any previously configured binding.
func (r *ServiceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Service account client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.ServiceAccounts()
	if err != nil {
		resp.Diagnostics.AddError("Service account client unavailable", "The configured SDK session did not expose service account services.")
	}
}

// IdentitySchema uses the stable nonzero object GUID for import and list-resource identity.
func (*ServiceAccountResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero service account object GUID."},
	}}
}

// ImportState adopts a verified GUID without recovering secret values or historical password-version metadata.
func (*ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	identity.ImportGUID(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password_wo_version"), int64(0))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("outbound_authentication_only"), false)...)
		if resp.Identity != nil {
			canonical, _ := identity.CanonicalGUID(req.ID)
			resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
}

// ModifyPlan separates explicit ownership from observations and preserves replacement-only creation inputs.
func (*ServiceAccountResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyServiceAccountPlan(ctx, req, resp)
}
