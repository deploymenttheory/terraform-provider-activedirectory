// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member_relationships

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

const DataSourceName = "activedirectory_group_member_relationships"

var (
	_ datasource.DataSource              = &GroupMemberRelationshipsDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupMemberRelationshipsDataSource{}
)

// GroupMemberRelationshipsDataSource discovers bounded direct links below one existing group GUID.
type GroupMemberRelationshipsDataSource struct{ client client.Memberships }

// NewGroupMemberRelationshipsDataSource constructs bounded read-only membership discovery without opening a connection.
func NewGroupMemberRelationshipsDataSource() datasource.DataSource {
	return &GroupMemberRelationshipsDataSource{}
}

// Metadata returns the registered direct-link collection data-source name.
func (*GroupMemberRelationshipsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's qualified session and clears a previous binding before reconfiguration.
func (d *GroupMemberRelationshipsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema separates the explicit group and bound from observed rows and completion status.
func (*GroupMemberRelationshipsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := map[string]schema.Attribute{}
	for name := range common.AttributeTypes() {
		nested[name] = schema.StringAttribute{Computed: true, Description: "Directory-observed direct-link " + name + "."}
	}
	resp.Schema = schema.Schema{
		Description: "Discover direct group-member relationships for one group. Does not expand nested or primary membership.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, Description: "Canonical group GUID identifying this scoped query."},
			"group_guid":    schema.StringAttribute{Required: true, Description: "Explicit existing group object GUID."},
			"result_limit":  schema.Int64Attribute{Optional: true, Description: "Maximum materialized direct links, from 1 to 10000; defaults to 1000."},
			"group_dn":      schema.StringAttribute{Computed: true, Description: "Current observed group distinguished name."},
			"complete":      schema.BoolAttribute{Computed: true, Description: "Whether bounded ranges and all member identities were completely observed."},
			"limit_reached": schema.BoolAttribute{Computed: true, Description: "Whether the configured bound stopped range retrieval."},
			"members": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Usable direct links sorted by member GUID.",
				NestedObject: schema.NestedAttributeObject{Attributes: nested},
			},
		},
	}
}
