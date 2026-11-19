// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_group"

var (
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithConfigure   = &GroupResource{}
	_ resource.ResourceWithIdentity    = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
	_ resource.ResourceWithModifyPlan  = &GroupResource{}
)

// GroupResource manages one group by GUID, preserving accepted effects and leaving membership to separate edge resources.
type GroupResource struct{ client client.Groups }

// NewGroupResource creates an unconfigured resource without opening a directory connection.
func NewGroupResource() resource.Resource { return &GroupResource{} }

// Metadata identifies the public group resource type.
func (*GroupResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema defines individual property ownership and semantic category/scope updates.
func (*GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = groupSchema()
}

// Configure binds the provider's qualified SDK session and retires any previously configured binding.
func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Group client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.Groups()
	if err != nil {
		resp.Diagnostics.AddError("Group client unavailable", "The configured SDK session did not expose group services.")
	}
}

// IdentitySchema uses the stable nonzero object GUID for import and list-resource identity.
func (*GroupResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero group object GUID."},
	}}
}

// ImportState adopts a verified GUID using its stable directory identity.
func (*GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
		if resp.Identity != nil {
			canonical, _ := identity.CanonicalGUID(req.ID)
			resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
}

// ModifyPlan distinguishes explicit property clears and refuses conflicting configured values.
func (*GroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyGroupPlan(ctx, req, resp)
}
