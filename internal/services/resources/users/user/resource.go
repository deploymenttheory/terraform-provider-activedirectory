// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_user"

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithIdentity    = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
	_ resource.ResourceWithModifyPlan  = &UserResource{}
)

// UserResource manages one user by GUID, preserving staged effects and never exposing a password in state.
type UserResource struct{ client client.Users }

// NewUserResource creates an unconfigured resource without opening a directory connection.
func NewUserResource() resource.Resource { return &UserResource{} }

// Metadata identifies the public user resource type.
func (*UserResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema defines typed property ownership and versioned write-only password input.
func (*UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = userSchema()
}

// Configure binds the provider's qualified SDK session and retires any previously configured binding.
func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("User client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.Users()
	if err != nil {
		resp.Diagnostics.AddError("User client unavailable", "The configured SDK session did not expose user services.")
	}
}

// IdentitySchema uses the stable nonzero object GUID for import and list-resource identity.
func (*UserResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero user object GUID."},
	}}
}

// ImportState adopts a verified GUID without recovering secret values or historical password-version metadata.
func (*UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
		if resp.Identity != nil {
			canonical, _ := identity.CanonicalGUID(req.ID)
			resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
}

// ModifyPlan distinguishes explicit property clears and refuses decreasing password-version intent.
func (*UserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyUserPlan(ctx, req, resp)
}
