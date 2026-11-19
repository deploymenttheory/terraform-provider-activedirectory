// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_accounts"
)

const DataSourceName = "activedirectory_service_account"

var (
	_ datasource.DataSource                   = &ServiceAccountDataSource{}
	_ datasource.DataSourceWithConfigure      = &ServiceAccountDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ServiceAccountDataSource{}
)

// ServiceAccountDataSource reads one service account by an explicit GUID or DN without owning directory attributes.
type ServiceAccountDataSource struct{ client client.ServiceAccounts }

// NewServiceAccountDataSource constructs an unconfigured single-service account data source.
func NewServiceAccountDataSource() datasource.DataSource { return &ServiceAccountDataSource{} }

// Metadata returns the registered single service account data source type.
func (*ServiceAccountDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Configure binds this read-only surface to the provider's existing authenticated SDK session.
func (d *ServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client, err = bundle.ServiceAccounts()
	if err != nil {
		resp.Diagnostics.AddError("Service account client unavailable", "The SDK session does not expose service account reads.")
	}
}

// Schema separates explicit lookup inputs from current directory observations.
func (*ServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := common.DataSourceAttributes()
	attributes["object_guid"] = schema.StringAttribute{Optional: true, Description: "Explicit object GUID. Supply exactly one of object_guid or dn."}
	attributes["dn"] = schema.StringAttribute{Optional: true, Description: "Explicit lookup distinguished name. Supply exactly one of object_guid or dn."}
	resp.Schema = schema.Schema{
		Description: "Reads one service account by GUID or DN. Missing or inaccessible objects report a diagnostic; no directory state is managed.",
		Attributes:  attributes,
	}
}

// ValidateConfig rejects conflicting/omitted lookup forms while allowing Terraform to resolve expressions.
func (*ServiceAccountDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	model, diagnostics := lookupModel(ctx, req.Config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || model.ObjectGUID.IsUnknown() || model.DN.IsUnknown() {
		return
	}
	_, diagnostics = common.ConstructLookup(model.ObjectGUID, model.DN)
	resp.Diagnostics.Append(diagnostics...)
}
