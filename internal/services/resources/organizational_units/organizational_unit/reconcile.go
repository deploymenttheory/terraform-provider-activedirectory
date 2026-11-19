// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

const pendingKey = "ad_ou_pending_mutation"

// pendingData is the Framework private-state boundary, also implementable by focused unit fixtures.
type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// pendingMutation stores directory intent only. Authentication configuration and credentials never enter it.
type pendingMutation struct {
	Version    int                `json:"version"`
	Operation  string             `json:"operation"`
	GUID       string             `json:"guid,omitempty"`
	Name       string             `json:"name,omitempty"`
	ParentDN   string             `json:"parentDn,omitempty"`
	Properties map[string]*string `json:"properties,omitempty"`
	Protected  *bool              `json:"protected,omitempty"`
}

func savePending(ctx context.Context, data pendingData, pending pendingMutation) diag.Diagnostics {
	if data == nil {
		return nil
	}
	pending.Version = 1
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("OU recovery state unavailable", "The provider could not preserve mutation intent. No new directory operation is authorized."),
		}
	}
	return data.SetKey(ctx, pendingKey, encoded)
}

func clearPending(ctx context.Context, data pendingData) diag.Diagnostics {
	if data == nil {
		return nil
	}
	return data.SetKey(ctx, pendingKey, nil)
}

func loadPending(ctx context.Context, data pendingData) (*pendingMutation, diag.Diagnostics) {
	if data == nil {
		return nil, nil
	}
	encoded, diagnostics := data.GetKey(ctx, pendingKey)
	if diagnostics.HasError() || len(encoded) == 0 {
		return nil, diagnostics
	}
	var pending pendingMutation
	if err := json.Unmarshal(encoded, &pending); err != nil || pending.Version != 1 {
		diagnostics.AddError("Invalid OU recovery state", "Stored mutation intent cannot be read. Reconcile the object independently before importing its verified GUID.")
		return nil, diagnostics
	}
	switch pending.Operation {
	case "create", "rename", "move", "update", "protection", "delete":
	default:
		diagnostics.AddError("Invalid OU recovery operation", "Stored mutation intent has an unknown operation. Reconcile independently before importing its verified GUID.")
		return nil, diagnostics
	}
	return &pending, diagnostics
}

// guardPending refuses mutation replay until a complete Read settles stored intent or an explicit import replaces it.
func guardPending(ctx context.Context, data pendingData) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, data)
	if pending != nil {
		diagnostics.AddError(
			"OU mutation needs reconciliation",
			"An earlier operation remains unresolved. Read must prove its intended effects before another write. Verify and import independently when proof is unavailable.",
		)
	}
	return diagnostics
}

// finishPending removes a checkpoint only for verified/acknowledged success or a known rejection without effects.
func finishPending(
	ctx context.Context,
	data pendingData,
	pending pendingMutation,
	state *OrganizationalUnitResourceModel,
	result shared.ObjectMutationResult,
	err error,
	success, aclEffects bool,
) diag.Diagnostics {
	if success || ((result.State == shared.ObjectWriteNotAttempted || result.State == shared.ObjectWriteRejected) &&
		!result.PartialEffectsPossible && !shared.IsUncertain(err) && !aclEffects) {
		return clearPending(ctx, data)
	}
	if !state.ID.IsNull() && !state.ID.IsUnknown() {
		pending.GUID = state.ID.ValueString()
	}
	return savePending(ctx, data, pending)
}

// reconcilePending clears private intent only when the complete SDK read proves every requested observable effect.
// Omission cannot prove an uncertain clear, and a DN-only candidate cannot prove an unobserved creation GUID.
func reconcilePending(ctx context.Context, data pendingData, unit *organizational_units.OrganizationalUnit) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, data)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	matched := pending.GUID != "" && unit != nil && unit.GUID != nil && unit.GUID.String() == pending.GUID && pending.Operation != "delete"
	// Terraform taints a failed Create even when the server accepted it. Automatic checkpoint
	// removal would allow the next apply to destroy/recreate that accepted object.
	if pending.Operation == "create" {
		matched = false
	}
	if pending.Name != "" {
		matched = matched && unit != nil && unit.Name != nil && *unit.Name == pending.Name
	}
	if pending.ParentDN != "" && unit != nil {
		parent, ok := observedParent(unit.DistinguishedName)
		matched = matched && ok && strings.EqualFold(parent, pending.ParentDN)
	}
	if pending.Protected != nil {
		matched = matched && unit != nil && unit.ProtectedFromAccidentalDeletion != nil && *unit.ProtectedFromAccidentalDeletion == *pending.Protected
	}
	for attribute, expected := range pending.Properties {
		if unit == nil || expected == nil {
			matched = false
			continue
		}
		observed := lifecycle.ObserveString(unit.Raw, attribute)
		matched = matched && observed.Usable && !observed.Diagnostics.HasError() && observed.Value.ValueString() == *expected
	}
	if matched {
		diagnostics.Append(clearPending(ctx, data)...)
		return diagnostics
	}
	diagnostics.AddError(
		"OU mutation still unresolved",
		"The read did not prove the pending effect. State and intent remain. No write replay is allowed; reconcile independently and import the verified GUID if needed.",
	)
	return diagnostics
}

func pendingProperties(model *OrganizationalUnitResourceModel) map[string]*string {
	properties := map[string]*string{}
	ignored := organizational_units.OUProperties{}
	for _, field := range propertyBindings(model, &ignored) {
		if field.value.IsNull() || field.value.IsUnknown() {
			continue
		}
		value := field.value.ValueString()
		properties[field.ldap] = &value
	}
	return properties
}

func pendingUpdate(request organizational_units.UpdateOrganizationalUnitRequest, model *OrganizationalUnitResourceModel) pendingMutation {
	properties := map[string]*string{}
	for _, field := range propertyBindings(model, &request.Properties) {
		if *field.target != nil {
			properties[field.ldap] = *field.target
		}
	}
	for _, change := range request.Changes {
		if change.Kind == shared.AttributeClear {
			properties[change.Attribute] = nil
		}
	}
	return pendingMutation{Operation: "update", GUID: model.ID.ValueString(), Properties: properties, Protected: request.ProtectedFromAccidentalDeletion}
}

func protectionEffects(result organizational_units.OUProtectionResult) bool {
	for _, step := range result.Steps {
		if step.State == shared.ObjectWriteAccepted || step.State == shared.ObjectWriteUncertain {
			return true
		}
	}
	return false
}
