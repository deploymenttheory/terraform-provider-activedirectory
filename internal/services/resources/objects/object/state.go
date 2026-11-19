// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// mapObservation updates only same-GUID observations; configured raw intent remains separate from returned bytes.
func mapObservation(model *ObjectResourceModel, object shared.Object, properties []string, acks acknowledgements) diag.Diagnostics {
	values, diagnostics := common.Project(object, properties)
	if diagnostics.HasError() {
		return diagnostics
	}
	id := values["id"].(types.String)
	if !model.ID.IsNull() && !model.ID.IsUnknown() && !model.ID.Equal(id) {
		diagnostics.AddError("Conflicting object readback identity", "A different GUID was returned. Existing identity and observations are retained.")
		return diagnostics
	}
	model.ID, model.Name = id, values["name"].(types.String)
	parent := values["parent_dn"].(types.String)
	if model.ParentDN.IsNull() || !strings.EqualFold(model.ParentDN.ValueString(), parent.ValueString()) {
		model.ParentDN = parent
	}
	if observed := values["object_class"].(types.String); !observed.IsNull() &&
		(model.ObjectClass.IsNull() || !strings.EqualFold(model.ObjectClass.ValueString(), observed.ValueString())) {
		model.ObjectClass = observed
	}
	model.DistinguishedName, model.ObjectClasses = values["distinguished_name"].(types.String), values["object_classes"].(types.List)
	model.ObservedAttributes = values["observed_attributes"].(types.Map)
	model.AttributeStatus, model.AttributeRanges = values["attribute_status"].(types.Map), values["attribute_ranges"].(types.Map)
	changes, _, construction := constructPlan(model, false)
	diagnostics.Append(construction...)
	if !construction.HasError() {
		model.NeedsUpdate = types.BoolValue(intentContradicted(changes, object, acks))
	}
	return diagnostics
}

// intentContradicted schedules reconciliation only from complete returned values, respecting unchanged accepted normalization.
func intentContradicted(changes []shared.AttributeChange, object shared.Object, acks acknowledgements) bool {
	for _, change := range changes {
		if acknowledgementMatches(acks, change, object) {
			continue
		}
		values := object.Values(change.Attribute)
		if values.State != shared.AttributePresent {
			continue
		}
		switch change.Kind {
		case shared.AttributeSet, shared.AttributeAddValues:
			if !positiveEffect(hashedChange(change), object) {
				return true
			}
		case shared.AttributeRemoveValues:
			for _, requested := range change.Values {
				if slices.ContainsFunc(values.Values, func(observed shared.RawValue) bool { return slices.Equal(observed.Bytes(), requested.Bytes()) }) {
					return true
				}
			}
		case shared.AttributeClear:
			if len(values.Values) != 0 {
				return true
			}
		}
	}
	return false
}

// effectiveChanges avoids known exact no-ops while retaining server-authoritative strict remove/clear semantics.
func effectiveChanges(changes []shared.AttributeChange, object shared.Object, acks acknowledgements) ([]shared.AttributeChange, []shared.AttributeChange, diag.Diagnostics) {
	var effective, full []shared.AttributeChange
	var diagnostics diag.Diagnostics
	for _, change := range changes {
		if acknowledgementMatches(acks, change, object) {
			continue
		}
		values := object.Values(change.Attribute)
		if values.State == shared.AttributePartial || values.State == shared.AttributeUnloaded {
			diagnostics.AddError("Incomplete raw attribute preflight", "An owned attribute is unloaded or ranged. No mutation phase is submitted from incomplete observations.")
			continue
		}
		if (change.Kind == shared.AttributeSet || change.Kind == shared.AttributeAddValues) && positiveEffect(hashedChange(change), object) {
			continue
		}
		original := change
		if change.Kind == shared.AttributeAddValues && values.State == shared.AttributePresent {
			change.Values = slices.DeleteFunc(slices.Clone(change.Values), func(requested shared.RawValue) bool {
				return slices.ContainsFunc(values.Values, func(observed shared.RawValue) bool { return slices.Equal(observed.Bytes(), requested.Bytes()) })
			})
		}
		effective, full = append(effective, change), append(full, original)
	}
	if diagnostics.HasError() {
		return nil, nil, diagnostics
	}
	return effective, full, diagnostics
}

// adoptIntent retains configured operation ownership only after its stage is accepted or its exact no-op is observed.
func adoptIntent(current, plan *ObjectResourceModel) {
	current.Attributes, current.AddValues, current.RemoveValues = plan.Attributes, plan.AddValues, plan.RemoveValues
	current.ClearAttributes, current.Properties = plan.ClearAttributes, plan.Properties
}

// persistState normalizes only unknown computed outputs and preserves any known GUID after partial failure.
func persistState(ctx context.Context, state *tfsdk.State, resourceIdentity *tfsdk.ResourceIdentity, model *ObjectResourceModel) diag.Diagnostics {
	for _, value := range []*types.String{&model.ID, &model.Name, &model.ParentDN, &model.ObjectClass, &model.DistinguishedName, &model.RecoveryDN} {
		if value.IsUnknown() {
			*value = types.StringNull()
		}
	}
	if model.ObjectClasses.IsUnknown() {
		model.ObjectClasses = types.ListNull(types.StringType)
	}
	for _, value := range []*types.Map{&model.ObservedAttributes, &model.AttributeRanges} {
		if value.IsUnknown() {
			*value = nullIntentMap()
		}
	}
	if model.AttributeStatus.IsUnknown() {
		model.AttributeStatus = types.MapNull(types.StringType)
	}
	if model.NeedsUpdate.IsUnknown() {
		model.NeedsUpdate = types.BoolNull()
	}
	diagnostics := state.Set(ctx, model)
	if resourceIdentity != nil && !model.ID.IsNull() {
		canonical, err := identity.CanonicalGUID(model.ID.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid object state identity", "Retained identity is not a nonzero GUID.")
		} else {
			diagnostics.Append(resourceIdentity.SetAttribute(ctx, path.Root("id"), canonical)...)
		}
	}
	return diagnostics
}
