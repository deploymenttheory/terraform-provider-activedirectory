// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

const DataSourceName = "activedirectory_organizational_units"

var (
	_ datasource.DataSource              = &OrganizationalUnitsDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationalUnitsDataSource{}
)

// OrganizationalUnitsDataSource returns bounded, deterministic OU observations without owning any object.
type OrganizationalUnitsDataSource struct{ client client.OrganizationalUnits }

// NewOrganizationalUnitsDataSource constructs an unconfigured scoped OU query.
func NewOrganizationalUnitsDataSource() datasource.DataSource {
	return &OrganizationalUnitsDataSource{}
}

// Metadata returns the registered plural organizational unit data source type.
func (*OrganizationalUnitsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's authenticated SDK session for scoped discovery.
func (d *OrganizationalUnitsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider clients", "Expected the configured Active Directory SDK bundle.")
		return
	}
	var err error
	d.client, err = bundle.OrganizationalUnits()
	if err != nil {
		resp.Diagnostics.AddError("OU client unavailable", "The SDK session does not expose organizational unit discovery.")
	}
}

// Schema requires an explicit base and distinguishes completed discovery from a caller-bounded result.
func (*OrganizationalUnitsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads organizational units below an explicit search base. Results are bounded, sorted by GUID and report completeness separately.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Deterministic query fingerprint; individual objects retain their GUID identities."},
			"base_dn": schema.StringAttribute{Required: true, Description: "Explicit distinguished name of the search base."},
			"ldap_filter": schema.StringAttribute{
				Optional:    true,
				Description: "Raw LDAP filter intersected with the SDK's OU class filter. PowerShell expressions are not parsed.",
			},
			"scope":     schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size": schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Zero never enables an unbounded query.",
			},
			"complete":      schema.BoolAttribute{Computed: true, Description: "Whether the SDK completed the entire selected search."},
			"limit_reached": schema.BoolAttribute{Computed: true, Description: "Whether the configured bound stopped retrieval; it does not prove further matches exist."},
			"organizational_units": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Usable OU observations sorted by GUID.",
				NestedObject: schema.NestedAttributeObject{Attributes: common.DataSourceAttributes()},
			},
		},
	}
}
