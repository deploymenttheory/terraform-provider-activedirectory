// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/computers"
)

const DataSourceName = "activedirectory_computers"

var (
	_ datasource.DataSource              = &ComputersDataSource{}
	_ datasource.DataSourceWithConfigure = &ComputersDataSource{}
)

// ComputersDataSource returns bounded, deterministic computer observations without owning any object.
type ComputersDataSource struct{ client client.Computers }

// NewComputersDataSource constructs an unconfigured scoped computer query.
func NewComputersDataSource() datasource.DataSource {
	return &ComputersDataSource{}
}

// Metadata returns the registered plural computer data source type.
func (*ComputersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's authenticated SDK session for scoped discovery.
func (d *ComputersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client, err = bundle.Computers()
	if err != nil {
		resp.Diagnostics.AddError("Computer client unavailable", "The SDK session does not expose computer discovery.")
	}
}

// Schema requires an explicit base and distinguishes completed discovery from a caller-bounded result.
func (*ComputersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads computers below an explicit search base. Results are bounded, sorted by GUID and report completeness separately.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Deterministic query fingerprint; individual objects retain their GUID identities."},
			"base_dn": schema.StringAttribute{Required: true, Description: "Explicit distinguished name of the search base."},
			"ldap_filter": schema.StringAttribute{
				Optional:    true,
				Description: "Raw LDAP filter intersected with the SDK's computer class filter. PowerShell expressions are not parsed.",
			},
			"scope":     schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size": schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Zero never enables an unbounded query.",
			},
			"complete":      schema.BoolAttribute{Computed: true, Description: "Whether the SDK completed the entire selected search."},
			"limit_reached": schema.BoolAttribute{Computed: true, Description: "Whether the configured bound stopped retrieval; it does not prove further matches exist."},
			"computers": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Usable computer observations sorted by GUID.",
				NestedObject: schema.NestedAttributeObject{Attributes: common.DataSourceAttributes()},
			},
		},
	}
}
