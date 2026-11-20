// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const DataSourceName = "activedirectory_group_members"

var (
	_ datasource.DataSource              = &GroupMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupMembersDataSource{}
)

// GroupMembersDataSource observes bounded direct or recursive membership without owning links or endpoint objects.
type GroupMembersDataSource struct{ client client.GroupMembershipQueries }

// NewGroupMembersDataSource constructs an unconfigured read-only query.
func NewGroupMembersDataSource() datasource.DataSource { return &GroupMembersDataSource{} }

// Metadata identifies the group-members data source.
func (*GroupMembersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's authenticated SDK session and retires the previous binding before reconfiguration.
func (d *GroupMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Group membership client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	d.client, err = bundle.GroupMembershipQueries()
	if err != nil {
		resp.Diagnostics.AddError("Group membership client unavailable", "The qualified SDK session does not expose group membership discovery.")
	}
}
