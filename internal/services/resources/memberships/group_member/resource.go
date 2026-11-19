// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_group_member"

var (
	_ resource.Resource                = &GroupMemberResource{}
	_ resource.ResourceWithConfigure   = &GroupMemberResource{}
	_ resource.ResourceWithIdentity    = &GroupMemberResource{}
	_ resource.ResourceWithImportState = &GroupMemberResource{}
)

// GroupMemberResource manages one direct link and retains unresolved mutation intent without replaying writes.
type GroupMemberResource struct{ client client.Memberships }

// NewGroupMemberResource constructs a resource without opening a directory connection.
func NewGroupMemberResource() resource.Resource { return &GroupMemberResource{} }

// Metadata identifies the direct-membership resource type.
func (*GroupMemberResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema separates the immutable GUID pair from computed directory names.
func (*GroupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = groupMemberSchema()
}

// Configure binds the resource to direct-membership operations on the provider's qualified SDK session.
func (r *GroupMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Membership client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.Memberships()
	if err != nil {
		resp.Diagnostics.AddError("Membership client unavailable", "The SDK session does not expose direct membership operations.")
	}
}

// IdentitySchema identifies the same ordered pair used by string import and list-resource discovery.
func (*GroupMemberResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Canonical group GUID/member GUID, in that order."},
	}}
}

// ImportState adopts one explicitly ordered pair without adding or removing directory links.
func (*GroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	group, member, diagnostics := identity.ParseRelationshipID(req.ID)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, _ := identity.RelationshipID(group, member)
	for name, value := range map[string]string{"id": id, "group_guid": group, "member_guid": member} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
	}
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
	}
}
