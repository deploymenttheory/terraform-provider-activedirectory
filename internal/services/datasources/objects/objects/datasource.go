// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

const DataSourceName = "activedirectory_objects"

var (
	_ datasource.DataSource              = &ObjectsDataSource{}
	_ datasource.DataSourceWithConfigure = &ObjectsDataSource{}
)

// ObjectsDataSource returns bounded, deterministic object observations without owning any object.
type ObjectsDataSource struct{ client client.Objects }

// NewObjectsDataSource constructs an unconfigured scoped object query.
func NewObjectsDataSource() datasource.DataSource {
	return &ObjectsDataSource{}
}

// Metadata returns the registered plural object data source type.
func (*ObjectsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's authenticated SDK session for scoped discovery.
func (d *ObjectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider clients", "Expected the configured Active Directory SDK bundle.")
		return
	}
	var err error
	d.client, err = bundle.Objects()
	if err != nil {
		resp.Diagnostics.AddError("Object client unavailable", "The SDK session does not expose object discovery.")
	}
}

// Schema requires an explicit base and distinguishes completed discovery from a caller-bounded result.
func (*ObjectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads objects below an explicit search base. Results are bounded, sorted by GUID and report completeness separately.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Deterministic query fingerprint; individual objects retain their GUID identities."},
			"base_dn": schema.StringAttribute{Required: true, Description: "Explicit distinguished name of the search base."},
			"ldap_filter": schema.StringAttribute{
				Optional:    true,
				Description: "Explicit raw LDAP filter; no object class is inferred. PowerShell expressions are not parsed.",
			},
			"scope":     schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size": schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Zero never enables an unbounded query.",
			},
			"properties": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Explicit raw LDAP display names to observe as base64 sets; identity fields are always selected.",
			},
			"complete":      schema.BoolAttribute{Computed: true, Description: "Whether the SDK completed the entire selected search."},
			"limit_reached": schema.BoolAttribute{Computed: true, Description: "Whether the configured bound stopped retrieval; it does not prove further matches exist."},
			"objects": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Usable object observations sorted by GUID.",
				NestedObject: schema.NestedAttributeObject{Attributes: common.DataSourceAttributes()},
			},
		},
	}
}
