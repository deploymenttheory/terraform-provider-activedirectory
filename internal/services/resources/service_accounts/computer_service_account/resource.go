// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_computer_service_account"

var (
	_ resource.Resource                = &ComputerServiceAccountResource{}
	_ resource.ResourceWithConfigure   = &ComputerServiceAccountResource{}
	_ resource.ResourceWithIdentity    = &ComputerServiceAccountResource{}
	_ resource.ResourceWithImportState = &ComputerServiceAccountResource{}
	_ resource.ResourceWithModifyPlan  = &ComputerServiceAccountResource{}
)

// ComputerServiceAccountResource manages one direct link and retains unresolved mutation intent without replaying writes.
type ComputerServiceAccountResource struct {
	client client.ComputerServiceAccountLinks
}

// NewComputerServiceAccountResource constructs a resource without opening a directory connection.
func NewComputerServiceAccountResource() resource.Resource { return &ComputerServiceAccountResource{} }

// Metadata identifies the direct-association resource type.
func (*ComputerServiceAccountResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema separates the immutable GUID pair from computed directory names.
func (*ComputerServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = computerServiceAccountSchema()
}

// Configure binds the resource to direct-association operations on the provider's qualified SDK session.
func (r *ComputerServiceAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Association client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.ComputerServiceAccountLinks()
	if err != nil {
		resp.Diagnostics.AddError("Association client unavailable", "The SDK session does not expose direct association operations.")
	}
}

// IdentitySchema identifies the same ordered pair used by string import and list-resource discovery.
func (*ComputerServiceAccountResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Canonical computer GUID/account GUID, in that order."},
	}}
}

// ModifyPlan keeps the configured desired link present while indeterminate refresh observations remain distinct in state.
func (*ComputerServiceAccountResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("is_associated"), types.BoolValue(true))...)
}

// ImportState adopts one explicitly ordered pair without adding or removing directory links.
func (*ComputerServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	computer, account, diagnostics := identity.ParseRelationshipID(req.ID)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, _ := identity.RelationshipID(computer, account)
	for name, value := range map[string]string{"id": id, "computer_guid": computer, "service_account_guid": account} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
	}
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
	}
}
