// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

const DataSourceName = "activedirectory_object"

var (
	_ datasource.DataSource                   = &ObjectDataSource{}
	_ datasource.DataSourceWithConfigure      = &ObjectDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ObjectDataSource{}
)

// ObjectDataSource reads one object by an explicit GUID or DN without owning directory attributes.
type ObjectDataSource struct{ client client.Objects }

// NewObjectDataSource constructs an unconfigured single-object data source.
func NewObjectDataSource() datasource.DataSource { return &ObjectDataSource{} }

// Metadata returns the registered single object data source type.
func (*ObjectDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure binds this read-only surface to the provider's existing authenticated SDK session.
func (d *ObjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("Object client unavailable", "The SDK session does not expose object reads.")
	}
}

// Schema separates explicit lookup inputs from current directory observations.
func (*ObjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := common.DataSourceAttributes()
	attributes["object_guid"] = schema.StringAttribute{Optional: true, Description: "Explicit object GUID. Supply exactly one of object_guid or dn."}
	attributes["dn"] = schema.StringAttribute{Optional: true, Description: "Explicit lookup distinguished name. Supply exactly one of object_guid or dn."}
	attributes["properties"] = schema.SetAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: "Explicit raw LDAP display names to observe as base64 sets; identity fields are always selected.",
	}
	resp.Schema = schema.Schema{
		Description: "Reads one object by GUID or DN. Missing or inaccessible objects report a diagnostic; no directory state is managed.",
		Attributes:  attributes,
	}
}

// ValidateConfig rejects conflicting/omitted lookup forms while allowing Terraform to resolve expressions.
func (*ObjectDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || model.ObjectGUID.IsUnknown() || model.DN.IsUnknown() || model.Properties.IsUnknown() {
		return
	}
	_, diagnostics = common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
	_, diagnostics = common.ConstructProperties(model.Properties)
	resp.Diagnostics.Append(diagnostics...)
}
