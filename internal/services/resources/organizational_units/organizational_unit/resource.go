// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

const ResourceName = "activedirectory_organizational_unit"

var (
	_ resource.Resource                = &OrganizationalUnitResource{}
	_ resource.ResourceWithConfigure   = &OrganizationalUnitResource{}
	_ resource.ResourceWithImportState = &OrganizationalUnitResource{}
	_ resource.ResourceWithIdentity    = &OrganizationalUnitResource{}
)

// OrganizationalUnitResource manages one OU and never recursively deletes child objects.
type OrganizationalUnitResource struct{ client client.OrganizationalUnits }

// NewOrganizationalUnitResource constructs an unconfigured OU resource.
func NewOrganizationalUnitResource() resource.Resource { return &OrganizationalUnitResource{} }

// Metadata returns the registered organizational unit resource type.
func (*OrganizationalUnitResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Configure binds the provider's shared SDK session without creating another authenticated connection.
func (r *OrganizationalUnitResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider clients", "Expected the configured Active Directory SDK client bundle.")
		return
	}
	var err error
	r.client, err = bundle.OrganizationalUnits()
	if err != nil {
		resp.Diagnostics.AddError("Missing organizational unit client", "The configured SDK session does not expose organizational unit services.")
	}
}

// Schema defines stable identity, mutable same-domain location and explicitly owned OU properties.
func (*OrganizationalUnitResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable object GUID and import identity.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":               schema.StringAttribute{Required: true, Description: "Literal OU name. Changes rename the same object GUID."},
		"parent_dn":          schema.StringAttribute{Required: true, Description: "Parent distinguished name. Changes move the OU within its domain."},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"recovery_dn":        schema.StringAttribute{Computed: true, Description: "Recovery locator retained when an accepted creation's GUID could not be observed."},
		"protected_from_accidental_deletion": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			Description: "Deletion protection, default true. Apply false explicitly before deleting or moving a protected OU. Destroy never disables protection automatically.",
		},
	}
	for name, description := range map[string]string{
		"description": "Directory description.", "display_name": "Display name.", "city": "City or locality.",
		"country": "Country code.", "state": "State or province.", "street_address": "Street address.",
		"postal_code": "Postal code.", "managed_by": "Explicit distinguished name of the managing object.",
	} {
		attributes[name] = schema.StringAttribute{Optional: true, Description: description}
	}
	resp.Schema = schema.Schema{
		Description: "Manages one Active Directory organizational unit using stable GUID identity. Deletion is always nonrecursive.",
		Attributes:  attributes,
	}
}

// IdentitySchema uses the same immutable GUID representation as import and list discovery.
func (*OrganizationalUnitResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero object GUID."},
	}}
}

// ImportState accepts a canonicalizable GUID or the equivalent Framework identity object.
func (*OrganizationalUnitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	identity.ImportGUID(ctx, req, resp)
	if !resp.Diagnostics.HasError() && resp.Identity != nil {
		canonical, _ := identity.CanonicalGUID(req.ID)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), canonical)...)
	}
}
