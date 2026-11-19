// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_computer"

var (
	_ resource.Resource                = &ComputerResource{}
	_ resource.ResourceWithConfigure   = &ComputerResource{}
	_ resource.ResourceWithIdentity    = &ComputerResource{}
	_ resource.ResourceWithImportState = &ComputerResource{}
	_ resource.ResourceWithModifyPlan  = &ComputerResource{}
)

// ComputerResource manages one computer by GUID, preserving staged effects and never exposing a password in state.
type ComputerResource struct{ client client.Computers }

// NewComputerResource creates an unconfigured resource without opening a directory connection.
func NewComputerResource() resource.Resource { return &ComputerResource{} }

// Metadata identifies the public computer resource type.
func (*ComputerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema defines typed property ownership and versioned write-only password input.
func (*ComputerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = computerSchema()
}

// Configure binds the provider's qualified SDK session and retires any previously configured binding.
func (r *ComputerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Computer client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.Computers()
	if err != nil {
		resp.Diagnostics.AddError("Computer client unavailable", "The configured SDK session did not expose computer services.")
	}
}

// IdentitySchema uses the stable nonzero object GUID for import and list-resource identity.
func (*ComputerResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero computer object GUID."},
	}}
}

// ImportState adopts a verified GUID without recovering secret values or historical password-version metadata.
func (*ComputerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
func (*ComputerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyComputerPlan(ctx, req, resp)
}
