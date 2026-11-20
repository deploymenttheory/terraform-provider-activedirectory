// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

const ListResourceName = "activedirectory_computer_service_account"

var (
	_ list.ListResource              = &ComputerServiceAccountListResource{}
	_ list.ListResourceWithConfigure = &ComputerServiceAccountListResource{}
)

// ComputerServiceAccountListResource discovers composite identities for individual direct association resources.
type ComputerServiceAccountListResource struct {
	client client.ComputerServiceAccountLinks
}

// NewComputerServiceAccountListResource constructs an unconfigured import-discovery surface.
func NewComputerServiceAccountListResource() list.ListResource {
	return &ComputerServiceAccountListResource{}
}

// Metadata returns the same type name as the registered managed computer-account resource.
func (*ComputerServiceAccountListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ListResourceName
}

// Configure binds discovery to the provider's shared SDK session rather than opening a connection per result.
func (r *ComputerServiceAccountListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client, err = bundle.ComputerServiceAccountLinks()
	if err != nil {
		resp.Diagnostics.AddError("Association client unavailable", "The SDK session does not expose direct association discovery.")
	}
}

// ListResourceConfigSchema requires one computer GUID and bounds association materialization before emitting imports.
func (*ComputerServiceAccountListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Discover directory computer/service-account links for composite-ID import. Host installation and password-reader permissions are outside this resource.",
		Attributes: map[string]schema.Attribute{
			"computer_guid": schema.StringAttribute{Required: true, Description: "Explicit existing computer object GUID."},
			"result_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum materialized links from 1 to 10000; defaults to 1000. Terraform's limit can lower this bound.",
			},
		},
	}
}
