// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

const pendingKey = "ad_group_pending_mutation"

type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// pendingMutation records only owned directory intent, never credentials, groupType replacement masks or member collections.
type pendingMutation struct {
	Version    int                `json:"version"`
	Operation  string             `json:"operation"`
	GUID       string             `json:"guid,omitempty"`
	Name       string             `json:"name,omitempty"`
	ParentDN   string             `json:"parentDn,omitempty"`
	Properties map[string]*string `json:"properties,omitempty"`
	Category   *string            `json:"category,omitempty"`
	Scope      *string            `json:"scope,omitempty"`
}

// savePending must preserve intent before submission; unavailable private storage never permits a write.
func savePending(ctx context.Context, storage pendingData, pending pendingMutation) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Group recovery storage unavailable", "Mutation intent cannot be preserved. No write was submitted.")}
	}
	pending.Version = 1
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Group recovery serialization failed", "Mutation intent cannot be preserved. No write was submitted.")}
	}
	return storage.SetKey(ctx, pendingKey, encoded)
}

func clearPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	if storage == nil {
		return nil
	}
	return storage.SetKey(ctx, pendingKey, nil)
}

func loadPending(ctx context.Context, storage pendingData) (*pendingMutation, diag.Diagnostics) {
	if storage == nil {
		return nil, nil
	}
	encoded, diagnostics := storage.GetKey(ctx, pendingKey)
	if diagnostics.HasError() || len(encoded) == 0 {
		return nil, diagnostics
	}
	var pending pendingMutation
	if err := json.Unmarshal(encoded, &pending); err != nil || pending.Version != 1 {
		diagnostics.AddError("Invalid group recovery state", "Stored mutation intent cannot be read. Reconcile independently before importing the verified GUID.")
		return nil, diagnostics
	}
	switch pending.Operation {
	case "create", "rename", "move", "update", "delete":
		return &pending, diagnostics
	default:
		diagnostics.AddError("Invalid group recovery operation", "Stored intent has an unknown operation. No further mutation is authorized.")
		return nil, diagnostics
	}
}

// guardPending prevents unresolved operations from being submitted again, including Terraform replacement of a tainted creation.
func guardPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if pending != nil {
		diagnostics.AddError(
			"Group mutation needs reconciliation",
			"Earlier intent remains unresolved. Verify its effects independently before another write or explicit GUID recovery.",
		)
	}
	return diagnostics
}

// finishPending clears acknowledged success or definite rejection without effects, retaining all other intent and known identity.
func finishPending(ctx context.Context, storage pendingData, pending pendingMutation, model *GroupResourceModel,
	result shared.ObjectMutationResult, err error, success bool,
) diag.Diagnostics {
	if success || (result.State == shared.ObjectWriteNotAttempted || result.State == shared.ObjectWriteRejected) && !result.PartialEffectsPossible && !shared.IsUncertain(err) {
		return clearPending(ctx, storage)
	}
	if !model.ID.IsNull() && !model.ID.IsUnknown() {
		pending.GUID = model.ID.ValueString()
	}
	return savePending(ctx, storage, pending)
}

// reconcilePending proves only observable intent. Failed Create stays blocked because Terraform taints it; omission cannot prove an uncertain clear.
func reconcilePending(ctx context.Context, storage pendingData, observed *groups.Group) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	matched := pending.Operation != "create" && pending.Operation != "delete" && observed != nil && observed.GUID != nil &&
		pending.GUID != "" && pending.GUID == observed.GUID.String()
	if pending.Name != "" {
		matched = matched && observed != nil && observed.Name != nil && pending.Name == *observed.Name
	}
	if pending.ParentDN != "" {
		parent, valid := "", false
		if observed != nil {
			parent, valid = identity.ParentDN(observed.DistinguishedName)
		}
		matched = matched && valid && strings.EqualFold(parent, pending.ParentDN)
	}
	matched = matched && pendingPropertiesObserved(pending, observed)
	if matched {
		diagnostics.Append(clearPending(ctx, storage)...)
	} else {
		diagnostics.AddError("Group mutation still unresolved", "Readback did not prove the intended effect. Known state and intent remain; no write replay is allowed.")
	}
	return diagnostics
}

// pendingPropertiesObserved compares the selected semantic bits and complete text values, preserving unrelated flags.
func pendingPropertiesObserved(pending *pendingMutation, observed *groups.Group) bool {
	if observed == nil {
		return false
	}
	if pending.Category != nil {
		value, diagnostics := groupCategory(types.StringValue(*pending.Category))
		if diagnostics.HasError() || observed.Category == nil || *observed.Category != value {
			return false
		}
	}
	if pending.Scope != nil {
		value, diagnostics := groupScope(types.StringValue(*pending.Scope))
		if diagnostics.HasError() || observed.Scope == nil || *observed.Scope != value {
			return false
		}
	}
	for name, expected := range pending.Properties {
		if expected == nil {
			return false
		}
		value := lifecycle.ObserveString(observed.Raw, name)
		if !value.Usable || value.Diagnostics.HasError() || value.Value.ValueString() != *expected {
			return false
		}
	}
	return true
}

// pendingForUpdate stores selected properties and category/scope intent without taking ownership of an entire groupType value.
func pendingForUpdate(request groups.UpdateGroupRequest, model *GroupResourceModel) pendingMutation {
	pending := pendingMutation{Operation: "update", GUID: model.ID.ValueString(), Properties: map[string]*string{}}
	for _, field := range propertyBindings(model, &request) {
		if *field.target != nil {
			pending.Properties[field.ldap] = *field.target
		}
	}
	for _, change := range request.Changes {
		if change.Kind == shared.AttributeClear {
			pending.Properties[change.Attribute] = nil
		}
	}
	if request.Category != nil {
		value := model.Category.ValueString()
		pending.Category = &value
	}
	if request.Scope != nil {
		value := model.Scope.ValueString()
		pending.Scope = &value
	}
	return pending
}
