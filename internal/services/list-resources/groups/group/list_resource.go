// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ListResourceName = "activedirectory_group"

var (
	_ list.ListResource              = &GroupListResource{}
	_ list.ListResourceWithConfigure = &GroupListResource{}
)

// GroupListResource discovers GUID identities for the corresponding managed group resource.
type GroupListResource struct{ client client.Groups }

// NewGroupListResource constructs an unconfigured import-discovery surface.
func NewGroupListResource() list.ListResource { return &GroupListResource{} }

// Metadata returns the same type name as the registered managed group resource.
func (*GroupListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ListResourceName
}

// Configure binds discovery to the provider's shared SDK session rather than opening a connection per result.
func (r *GroupListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client, err = bundle.Groups()
	if err != nil {
		resp.Diagnostics.AddError("Group client unavailable", "The SDK session does not expose group discovery.")
	}
}

// ListResourceConfigSchema requires an explicit scope and bounded materialization before identities are emitted.
func (*GroupListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discovers groups for GUID import. The SDK materializes a bounded query; this is not an unbounded LDAP page stream.",
		Attributes: map[string]schema.Attribute{
			"base_dn":     schema.StringAttribute{Required: true, Description: "Explicit search-base distinguished name."},
			"ldap_filter": schema.StringAttribute{Optional: true, Description: "Raw LDAP filter intersected with the SDK's group class filter."},
			"scope":       schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size":   schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Terraform's limit can lower this bound.",
			},
		},
	}
}
