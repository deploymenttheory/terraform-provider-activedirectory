// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const DataSourceName = "activedirectory_computer_service_account"

var (
	_ datasource.DataSource              = &ComputerServiceAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &ComputerServiceAccountDataSource{}
)

// ComputerServiceAccountDataSource observes one existing direct computer/account link by stable endpoint GUIDs.
type ComputerServiceAccountDataSource struct {
	client client.ComputerServiceAccountLinks
}

// NewComputerServiceAccountDataSource constructs a read-only direct-link lookup without opening a connection.
func NewComputerServiceAccountDataSource() datasource.DataSource {
	return &ComputerServiceAccountDataSource{}
}

// Metadata returns the registered direct-link data-source name.
func (*ComputerServiceAccountDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure reuses the provider's qualified session and clears a previous binding before reconfiguration.
func (d *ComputerServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Schema distinguishes explicit GUID lookup inputs from computed link identity and current names.
func (*ComputerServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read one direct computer-account link. Primary and nested association do not satisfy this lookup.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, Description: "Canonical ordered computer GUID/account GUID identity."},
			"computer_guid":        schema.StringAttribute{Required: true, Description: "Existing computer object GUID."},
			"service_account_guid": schema.StringAttribute{Required: true, Description: "Existing direct account object GUID."},
			"computer_dn":          schema.StringAttribute{Computed: true, Description: "Current directory-observed computer name."},
			"service_account_dn":   schema.StringAttribute{Computed: true, Description: "Current directory-observed account name."},
		},
	}
}
