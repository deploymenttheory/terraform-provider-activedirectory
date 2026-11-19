// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

const DataSourceName = "activedirectory_organizational_unit"

var (
	_ datasource.DataSource                   = &OrganizationalUnitDataSource{}
	_ datasource.DataSourceWithConfigure      = &OrganizationalUnitDataSource{}
	_ datasource.DataSourceWithValidateConfig = &OrganizationalUnitDataSource{}
)

// OrganizationalUnitDataSource reads one OU by an explicit GUID or DN without owning directory attributes.
type OrganizationalUnitDataSource struct{ client client.OrganizationalUnits }

// NewOrganizationalUnitDataSource constructs an unconfigured single-OU data source.
func NewOrganizationalUnitDataSource() datasource.DataSource { return &OrganizationalUnitDataSource{} }

// Metadata returns the registered single organizational unit data source type.
func (*OrganizationalUnitDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure binds this read-only surface to the provider's existing authenticated SDK session.
func (d *OrganizationalUnitDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("OU client unavailable", "The SDK session does not expose organizational unit reads.")
	}
}

// Schema separates explicit lookup inputs from current directory observations.
func (*OrganizationalUnitDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := common.DataSourceAttributes()
	attributes["object_guid"] = schema.StringAttribute{Optional: true, Description: "Explicit object GUID. Supply exactly one of object_guid or dn."}
	attributes["dn"] = schema.StringAttribute{Optional: true, Description: "Explicit lookup distinguished name. Supply exactly one of object_guid or dn."}
	resp.Schema = schema.Schema{
		Description: "Reads one organizational unit by GUID or DN. Missing or inaccessible objects report a diagnostic; no directory state is managed.",
		Attributes:  attributes,
	}
}

// ValidateConfig rejects conflicting/omitted lookup forms while allowing Terraform to resolve expressions.
func (*OrganizationalUnitDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || model.ObjectGUID.IsUnknown() || model.DN.IsUnknown() {
		return
	}
	_, diagnostics = common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
}
