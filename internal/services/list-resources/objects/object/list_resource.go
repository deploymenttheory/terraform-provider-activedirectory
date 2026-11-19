// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ListResourceName = "activedirectory_object"

var (
	_ list.ListResource              = &ObjectListResource{}
	_ list.ListResourceWithConfigure = &ObjectListResource{}
)

// ObjectListResource discovers GUID identities for the corresponding managed object resource.
type ObjectListResource struct{ client client.Objects }

// NewObjectListResource constructs an unconfigured import-discovery surface.
func NewObjectListResource() list.ListResource { return &ObjectListResource{} }

// Metadata returns the same type name as the registered managed object resource.
func (*ObjectListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ListResourceName
}

// Configure binds discovery to the provider's shared SDK session rather than opening a connection per result.
func (r *ObjectListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client, err = bundle.Objects()
	if err != nil {
		resp.Diagnostics.AddError("Object client unavailable", "The SDK session does not expose object discovery.")
	}
}

// ListResourceConfigSchema requires an explicit scope and bounded materialization before identities are emitted.
func (*ObjectListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discovers objects for GUID import. The SDK materializes a bounded query; this is not an unbounded LDAP page stream.",
		Attributes: map[string]schema.Attribute{
			"base_dn":     schema.StringAttribute{Required: true, Description: "Explicit search-base distinguished name."},
			"ldap_filter": schema.StringAttribute{Optional: true, Description: "Explicit raw LDAP filter; no object class is inferred."},
			"properties": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Explicit raw LDAP display names to observe without taking raw attribute ownership in generated resources.",
			},
			"scope":     schema.StringAttribute{Optional: true, Description: "base, one_level or subtree; defaults to subtree."},
			"page_size": schema.Int64Attribute{Optional: true, Description: "LDAP page size from 1 to 1000; defaults to 256."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized objects from 1 to 10000; defaults to 1000. Terraform's limit can lower this bound.",
			},
		},
	}
}
