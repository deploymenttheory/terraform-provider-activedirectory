// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const DataSourceName = "activedirectory_group_member"

var (
	_ datasource.DataSource              = &GroupMemberDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupMemberDataSource{}
)

// GroupMemberDataSource observes one existing direct group/member link by stable endpoint GUIDs.
type GroupMemberDataSource struct{ client client.Memberships }

// NewGroupMemberDataSource constructs a read-only direct-link lookup without opening a connection.
func NewGroupMemberDataSource() datasource.DataSource { return &GroupMemberDataSource{} }

// Metadata returns the registered direct-link data-source name.
func (*GroupMemberDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's qualified session and clears a previous binding before reconfiguration.
func (d *GroupMemberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Membership client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	d.client, err = bundle.Memberships()
	if err != nil {
		resp.Diagnostics.AddError("Membership client unavailable", "The SDK session does not expose direct membership reads.")
	}
}

// Schema distinguishes explicit GUID lookup inputs from computed link identity and current names.
func (*GroupMemberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read one direct group-member link. Primary and nested membership do not satisfy this lookup.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "Canonical ordered group GUID/member GUID identity."},
			"group_guid":  schema.StringAttribute{Required: true, Description: "Existing group object GUID."},
			"member_guid": schema.StringAttribute{Required: true, Description: "Existing direct member object GUID."},
			"group_dn":    schema.StringAttribute{Computed: true, Description: "Current directory-observed group name."},
			"member_dn":   schema.StringAttribute{Computed: true, Description: "Current directory-observed member name."},
		},
	}
}
