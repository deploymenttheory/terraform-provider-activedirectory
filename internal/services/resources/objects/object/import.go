// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// importSelection adds explicit attribute ownership to GUID import without requesting wildcard values or guessing caller intent.
type importSelection struct {
	ID         string   `json:"id"`
	Attributes []string `json:"attributes,omitempty"`
	Properties []string `json:"properties,omitempty"`
}

// ImportState reads an existing GUID only. Optional JSON selectors adopt named visible raw attributes without submitting writes.
func (r *ObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.configured(&resp.Diagnostics) {
		return
	}
	id, err := importIdentity(ctx, req)
	if err != nil {
		resp.Diagnostics.AddError("Invalid object import identity", "A known nonzero GUID is required.")
		return
	}
	selection := importSelection{ID: id}
	if strings.HasPrefix(strings.TrimSpace(id), "{") {
		decoder := json.NewDecoder(strings.NewReader(id))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&selection); decodeErr != nil {
			resp.Diagnostics.AddError("Invalid object import selection", "Use a GUID or JSON containing id, optional attributes and optional properties arrays.")
			return
		}
		if !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
			resp.Diagnostics.AddError("Invalid object import selection", "Import accepts one JSON object only.")
			return
		}
	}
	canonical, err := identity.CanonicalGUID(selection.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid object import GUID", "Import requires the existing nonzero object GUID.")
		return
	}
	intent := common.AttributeIntent{Set: map[string][]string{}}
	for _, name := range selection.Attributes {
		if _, duplicate := intent.Set[name]; duplicate {
			resp.Diagnostics.AddError("Duplicate import attribute", "Each selected owned attribute must appear once.")
			return
		}
		intent.Set[name] = []string{""}
	}
	if _, err := common.ConstructChanges(intent); err != nil {
		resp.Diagnostics.AddError("Invalid import attribute ownership", "Use distinct writable LDAP display names; identity, class and naming fields have separate ownership.")
		return
	}
	properties := make([]attr.Value, 0, len(selection.Properties))
	for _, name := range selection.Properties {
		properties = append(properties, types.StringValue(name))
	}
	model := emptyModel()
	model.ID, model.Properties = types.StringValue(canonical), types.SetValueMust(types.StringType, properties)
	if len(properties) == 0 {
		model.Properties = types.SetNull(types.StringType)
	}
	selected := append([]attr.Value{}, properties...)
	for _, name := range selection.Attributes {
		selected = append(selected, types.StringValue(name))
	}
	requested, diagnostics := common.ConstructProperties(types.SetValueMust(types.StringType, selected))
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	object, diagnostics := r.observe(ctx, model.ID, requested)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || object == nil {
		return
	}
	owned := map[string]attr.Value{}
	for _, name := range selection.Attributes {
		values := object.Values(name)
		if values.State != shared.AttributePresent || len(values.Values) == 0 {
			resp.Diagnostics.AddError("Import attribute unavailable", "An explicitly owned raw attribute is absent, hidden or incomplete. No empty ownership is invented.")
			continue
		}
		owned[name] = rawSet(values.Values)
	}
	if len(owned) != 0 {
		model.Attributes = types.MapValueMust(common.RawSetType(), owned)
	}
	resp.Diagnostics.Append(mapObservation(&model, *object, requested, nil)...)
	if model.ObjectClass.IsNull() {
		resp.Diagnostics.AddError("Import class unavailable", "The instantiated class was not observed; import cannot guess a required class.")
	}
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(persistState(ctx, &resp.State, resp.Identity, &model)...)
	}
}

// emptyModel initializes typed nulls for import and isolated state tests without claiming absent raw values are empty sets.
func emptyModel() ObjectResourceModel {
	return ObjectResourceModel{
		ID: types.StringNull(), Name: types.StringNull(), ParentDN: types.StringNull(), ObjectClass: types.StringNull(),
		DistinguishedName: types.StringNull(), ObjectClasses: types.ListNull(types.StringType), RecoveryDN: types.StringNull(),
		Attributes: nullIntentMap(), AddValues: nullIntentMap(), RemoveValues: nullIntentMap(),
		ClearAttributes: types.SetNull(types.StringType), Properties: types.SetNull(types.StringType),
		ObservedAttributes: nullIntentMap(), AttributeStatus: types.MapNull(types.StringType), AttributeRanges: nullIntentMap(), NeedsUpdate: types.BoolValue(false),
	}
}
