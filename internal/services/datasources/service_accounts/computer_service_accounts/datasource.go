// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_accounts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

const DataSourceName = "activedirectory_computer_service_accounts"

var (
	_ datasource.DataSource              = &ComputerServiceAccountsDataSource{}
	_ datasource.DataSourceWithConfigure = &ComputerServiceAccountsDataSource{}
)

// ComputerServiceAccountsDataSource discovers bounded direct links below one existing computer GUID.
type ComputerServiceAccountsDataSource struct {
	client client.ComputerServiceAccountLinks
}

// NewComputerServiceAccountsDataSource constructs bounded read-only association discovery without opening a connection.
func NewComputerServiceAccountsDataSource() datasource.DataSource {
	return &ComputerServiceAccountsDataSource{}
}

// Metadata returns the registered direct-link collection data-source name.
func (*ComputerServiceAccountsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's qualified session and clears a previous binding before reconfiguration.
func (d *ComputerServiceAccountsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Association client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	d.client, err = bundle.ComputerServiceAccountLinks()
	if err != nil {
		resp.Diagnostics.AddError("Association client unavailable", "The SDK session does not expose direct association reads.")
	}
}

// Schema separates the explicit computer and bound from observed rows and completion status.
func (*ComputerServiceAccountsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := map[string]schema.Attribute{}
	for name := range common.AttributeTypes() {
		nested[name] = schema.StringAttribute{Computed: true, Description: "Directory-observed direct-link " + name + "."}
	}
	resp.Schema = schema.Schema{
		Description: "Discover managed service accounts associated with one computer in the directory. No local installation or password retrieval is performed.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, Description: "Canonical computer GUID identifying this scoped query."},
			"computer_guid": schema.StringAttribute{Required: true, Description: "Explicit existing computer object GUID."},
			"result_limit":  schema.Int64Attribute{Optional: true, Description: "Maximum materialized direct links, from 1 to 10000; defaults to 1000."},
			"computer_dn":   schema.StringAttribute{Computed: true, Description: "Current observed computer distinguished name."},
			"complete":      schema.BoolAttribute{Computed: true, Description: "Whether bounded ranges and all account identities were completely observed."},
			"limit_reached": schema.BoolAttribute{Computed: true, Description: "Whether the configured bound stopped range retrieval."},
			"service_accounts": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Usable direct links sorted by account GUID.",
				NestedObject: schema.NestedAttributeObject{Attributes: nested},
			},
		},
	}
}
