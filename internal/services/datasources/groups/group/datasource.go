// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/groups"
)

const DataSourceName = "activedirectory_group"

var (
	_ datasource.DataSource                   = &GroupDataSource{}
	_ datasource.DataSourceWithConfigure      = &GroupDataSource{}
	_ datasource.DataSourceWithValidateConfig = &GroupDataSource{}
)

// GroupDataSource reads one group by an explicit GUID or DN without owning directory attributes.
type GroupDataSource struct{ client client.Groups }

// NewGroupDataSource constructs an unconfigured single-group data source.
func NewGroupDataSource() datasource.DataSource { return &GroupDataSource{} }

// Metadata returns the registered single group data source type.
func (*GroupDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure binds this read-only surface to the provider's existing authenticated SDK session.
func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client, err = bundle.Groups()
	if err != nil {
		resp.Diagnostics.AddError("Group client unavailable", "The SDK session does not expose group reads.")
	}
}

// Schema separates explicit lookup inputs from current directory observations.
func (*GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := common.DataSourceAttributes()
	attributes["object_guid"] = schema.StringAttribute{Optional: true, Description: "Explicit object GUID. Supply exactly one of object_guid or dn."}
	attributes["dn"] = schema.StringAttribute{Optional: true, Description: "Explicit lookup distinguished name. Supply exactly one of object_guid or dn."}
	resp.Schema = schema.Schema{
		Description: "Reads one group by GUID or DN. Missing or inaccessible objects report a diagnostic; no directory state is managed.",
		Attributes:  attributes,
	}
}

// ValidateConfig rejects conflicting/omitted lookup forms while allowing Terraform to resolve expressions.
func (*GroupDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || model.ObjectGUID.IsUnknown() || model.DN.IsUnknown() {
		return
	}
	_, diagnostics = common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
}
