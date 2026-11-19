// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ListResourceName = "activedirectory_group_member"

var (
	_ list.ListResource              = &GroupMemberListResource{}
	_ list.ListResourceWithConfigure = &GroupMemberListResource{}
)

// GroupMemberListResource discovers composite identities for individual direct membership resources.
type GroupMemberListResource struct{ client client.Memberships }

// NewGroupMemberListResource constructs an unconfigured import-discovery surface.
func NewGroupMemberListResource() list.ListResource { return &GroupMemberListResource{} }

// Metadata returns the same type name as the registered managed group-member resource.
func (*GroupMemberListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ListResourceName
}

// Configure binds discovery to the provider's shared SDK session rather than opening a connection per result.
func (r *GroupMemberListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider clients", "Expected the configured Active Directory SDK bundle.")
		return
	}
	var err error
	r.client, err = bundle.Memberships()
	if err != nil {
		resp.Diagnostics.AddError("Membership client unavailable", "The SDK session does not expose direct membership discovery.")
	}
}

// ListResourceConfigSchema requires one group GUID and bounds membership materialization before emitting imports.
func (*GroupMemberListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discover direct group-member links for composite-ID import. Primary and nested memberships are excluded.",
		Attributes: map[string]schema.Attribute{
			"group_guid": schema.StringAttribute{Required: true, Description: "Explicit existing group object GUID."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized links from 1 to 10000; defaults to 1000. Terraform's limit can lower this bound.",
			},
		},
	}
}
