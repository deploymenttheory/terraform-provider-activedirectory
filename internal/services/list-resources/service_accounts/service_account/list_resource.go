// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ListResourceName = "activedirectory_service_account"

var (
	_ list.ListResource              = &ServiceAccountListResource{}
	_ list.ListResourceWithConfigure = &ServiceAccountListResource{}
)

// ServiceAccountListResource discovers GUID identities for the corresponding managed service account resource.
type ServiceAccountListResource struct{ client client.ServiceAccounts }

// NewServiceAccountListResource constructs an unconfigured import-discovery surface.
func NewServiceAccountListResource() list.ListResource { return &ServiceAccountListResource{} }

// Metadata returns the same type name as the registered managed service account resource.
func (*ServiceAccountListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ListResourceName
}

// Configure binds discovery to the provider's shared SDK session rather than opening a connection per result.
func (r *ServiceAccountListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider clients", "Expected the configured Active Directory SDK bundle.")
		return
	}
	var err error
	r.client, err = bundle.ServiceAccounts()
	if err != nil {
		resp.Diagnostics.AddError("Service account client unavailable", "The SDK session does not expose service_account discovery.")
	}
}

// ListResourceConfigSchema requires an explicit scope and bounded materialization before identities are emitted.
func (*ServiceAccountListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discovers service accounts for GUID import. The SDK materializes a bounded query; this is not an unbounded LDAP page stream.",
		Attributes: map[string]schema.Attribute{
			"base_dn":     schema.StringAttribute{Required: true, Description: "Explicit search-base distinguished name."},
			"ldap_filter": schema.StringAttribute{Optional: true, Description: "Raw LDAP filter intersected with the SDK's sMSA/gMSA class filter."},
			"scope":       schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size":   schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Terraform's limit can lower this bound.",
			},
		},
	}
}
