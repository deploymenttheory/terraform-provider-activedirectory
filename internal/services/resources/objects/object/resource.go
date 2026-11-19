// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

const ResourceName = "activedirectory_object"

var (
	_ resource.Resource                = &ObjectResource{}
	_ resource.ResourceWithConfigure   = &ObjectResource{}
	_ resource.ResourceWithIdentity    = &ObjectResource{}
	_ resource.ResourceWithImportState = &ObjectResource{}
	_ resource.ResourceWithModifyPlan  = &ObjectResource{}
)

// ObjectResource manages a generic object through a shared qualified SDK session, retaining uncertain effects by GUID.
type ObjectResource struct{ client client.Objects }

// NewObjectResource constructs an unconfigured resource without opening a directory connection.
func NewObjectResource() resource.Resource { return &ObjectResource{} }

// Metadata identifies the public generic object resource type.
func (*ObjectResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceName
}

// Schema exposes raw intent and observations independently so directory normalization never invents empty state.
func (*ObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = objectSchema()
}

// Configure binds all object operations to the provider's existing authenticated session.
func (r *ObjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = nil
	if req.ProviderData == nil {
		return
	}
	bundle, ok := req.ProviderData.(*client.Bundle)
	if !ok || bundle == nil {
		resp.Diagnostics.AddError("Object client unavailable", "A configured Active Directory SDK bundle is required.")
		return
	}
	var err error
	r.client, err = bundle.Objects()
	if err != nil {
		resp.Diagnostics.AddError("Object client unavailable", "The SDK session did not expose generic object operations.")
	}
}

// IdentitySchema uses only the stable GUID, independent of class selection, property ownership or current location.
func (*ObjectResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true, Description: "Stable nonzero object GUID."},
	}}
}

// ModifyPlan schedules updates for positively observed drift without rewriting configured raw intent.
func (*ObjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inputs := []attr.Value{plan.Name, plan.ParentDN, plan.ObjectClass, plan.Attributes, plan.AddValues, plan.RemoveValues, plan.ClearAttributes, plan.Properties}
	if allKnown(inputs) {
		_, _, diagnostics := constructPlan(&plan, req.State.Raw.IsNull())
		resp.Diagnostics.Append(diagnostics...)
	}
	if !req.State.Raw.IsNull() {
		var current ObjectResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
		if current.NeedsUpdate.ValueBool() {
			plan.DistinguishedName, plan.ObjectClasses = types.StringUnknown(), types.ListUnknown(types.StringType)
			plan.ObservedAttributes, plan.AttributeRanges = types.MapUnknown(common.RawSetType()), types.MapUnknown(common.RawSetType())
			plan.AttributeStatus = types.MapUnknown(types.StringType)
		}
	}
	plan.NeedsUpdate = types.BoolValue(false)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// allKnown allows unresolved Terraform expressions during planning while requiring complete apply-time validation.
func allKnown(values []attr.Value) bool {
	for _, value := range values {
		if value.IsUnknown() {
			return false
		}
		switch collection := value.(type) {
		case types.Map:
			for _, element := range collection.Elements() {
				if !allKnown([]attr.Value{element}) {
					return false
				}
			}
		case types.Set:
			if !allKnown(collection.Elements()) {
				return false
			}
		}
	}
	return true
}

// importIdentity reads the explicit GUID from Terraform's identity import form.
func importIdentity(ctx context.Context, req resource.ImportStateRequest) (string, error) {
	if req.ID != "" || req.Identity == nil {
		return req.ID, nil
	}
	var value types.String
	if diagnostics := req.Identity.GetAttribute(ctx, path.Root("id"), &value); diagnostics.HasError() || value.IsUnknown() || value.IsNull() {
		return "", client.ErrObjects
	}
	return value.ValueString(), nil
}
