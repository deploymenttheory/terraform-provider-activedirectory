// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/computers"
)

const DataSourceName = "activedirectory_computer"

var (
	_ datasource.DataSource                   = &ComputerDataSource{}
	_ datasource.DataSourceWithConfigure      = &ComputerDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ComputerDataSource{}
)

// ComputerDataSource reads one computer by an explicit GUID or DN without owning directory attributes.
type ComputerDataSource struct{ client client.Computers }

// NewComputerDataSource constructs an unconfigured single-computer data source.
func NewComputerDataSource() datasource.DataSource { return &ComputerDataSource{} }

// Metadata returns the registered single computer data source type.
func (*ComputerDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure binds this read-only surface to the provider's existing authenticated SDK session.
func (d *ComputerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("Computer client unavailable", "The SDK session does not expose computer reads.")
	}
}

// Schema separates explicit lookup inputs from current directory observations.
func (*ComputerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := common.DataSourceAttributes()
	attributes["object_guid"] = schema.StringAttribute{Optional: true, Description: "Explicit object GUID. Supply exactly one of object_guid or dn."}
	attributes["dn"] = schema.StringAttribute{Optional: true, Description: "Explicit lookup distinguished name. Supply exactly one of object_guid or dn."}
	resp.Schema = schema.Schema{
		Description: "Reads one computer by GUID or DN. Missing or inaccessible objects report a diagnostic; no directory state is managed.",
		Attributes:  attributes,
	}
}

// ValidateConfig rejects conflicting/omitted lookup forms while allowing Terraform to resolve expressions.
func (*ComputerDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || model.ObjectGUID.IsUnknown() || model.DN.IsUnknown() {
		return
	}
	_, diagnostics = common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
}
