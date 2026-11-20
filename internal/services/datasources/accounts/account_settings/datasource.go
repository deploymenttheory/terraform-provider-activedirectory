// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/account_settings"
)

const DataSourceName = "activedirectory_account_settings"

var (
	_ datasource.DataSource                   = &AccountSettingsDataSource{}
	_ datasource.DataSourceWithConfigure      = &AccountSettingsDataSource{}
	_ datasource.DataSourceWithValidateConfig = &AccountSettingsDataSource{}
)

// AccountSettingsDataSource observes one existing account's settings without owning or changing them.
type AccountSettingsDataSource struct{ client client.AccountSettings }

// NewAccountSettingsDataSource constructs an unconfigured read-only surface without connecting to AD.
func NewAccountSettingsDataSource() datasource.DataSource { return &AccountSettingsDataSource{} }

// Metadata identifies the public single-account settings data source.
func (*AccountSettingsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = DataSourceName
}

// Schema separates explicit lookup inputs from nullable typed directory observations.
func (*AccountSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"object_guid": schema.StringAttribute{Optional: true, Description: "Explicit existing account GUID; select exactly one of object_guid or dn."},
		"dn":          schema.StringAttribute{Optional: true, Description: "Explicit existing account distinguished name; select exactly one of object_guid or dn."},
	}
	for name, kind := range common.AttributeTypes() {
		description := "Observed " + name + " on the selected account and server; unavailable metadata remains null. This data source never writes settings."
		switch kind {
		case types.StringType:
			attributes[name] = schema.StringAttribute{Computed: true, Description: description}
		case types.BoolType:
			attributes[name] = schema.BoolAttribute{Computed: true, Description: description}
		case types.Int64Type:
			attributes[name] = schema.Int64Attribute{Computed: true, Description: description}
		}
	}
	resp.Schema = schema.Schema{
		Description: "Read existing account-control, expiration and computed status by explicit GUID or DN. Later reads are pinned " +
			"to the resolved GUID; no account is created, changed or owned.",
		Attributes: attributes,
	}
}

// Configure binds the same qualified SDK session used by other provider surfaces and retires any previous binding.
func (d *AccountSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Account settings client unavailable", "A configured Active Directory client bundle is required.")
		return
	}
	var err error
	d.client, err = bundle.AccountSettings()
	if err != nil {
		resp.Diagnostics.AddError("Account settings client unavailable", "The qualified SDK session did not expose account settings reads.")
	}
}

// ValidateConfig rejects known conflicting or omitted lookup forms while allowing Terraform expressions to resolve.
func (*AccountSettingsDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var guid, dn types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("object_guid"), &guid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("dn"), &dn)...)
	if resp.Diagnostics.HasError() || guid.IsUnknown() || dn.IsUnknown() {
		return
	}
	_, diagnostics := common.ConstructLookup(guid, dn)
	resp.Diagnostics.Append(diagnostics...)
}
