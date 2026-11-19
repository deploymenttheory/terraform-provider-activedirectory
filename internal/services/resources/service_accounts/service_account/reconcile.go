// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

const pendingKey = "ad_service_account_pending_mutation"

type pendingData interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

// pendingMutation deliberately contains no password value, hash, provider configuration or credentials.
type pendingMutation struct {
	Version         int                `json:"version"`
	Operation       string             `json:"operation"`
	GUID            string             `json:"guid,omitempty"`
	Name            string             `json:"name,omitempty"`
	ParentDN        string             `json:"parentDn,omitempty"`
	Properties      map[string]*string `json:"properties,omitempty"`
	Flags           map[string]bool    `json:"flags,omitempty"`
	Expiration      *int64             `json:"expiration,omitempty"`
	PasswordVersion int64              `json:"passwordVersion,omitempty"`
	Encryption      *uint32            `json:"encryption,omitempty"`
	Compound        *bool              `json:"compound,omitempty"`
	SPNs            *[]string          `json:"spns,omitempty"`
	Readers         *[]string          `json:"readers,omitempty"`
}

// savePending must succeed before a write; missing private storage is a refusal rather than permission to lose intent.
func savePending(ctx context.Context, storage pendingData, pending pendingMutation) diag.Diagnostics {
	if storage == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Service account recovery storage unavailable", "Mutation intent cannot be preserved. No write was submitted.")}
	}
	pending.Version = 1
	encoded, err := json.Marshal(pending)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Service account recovery serialization failed", "Mutation intent cannot be preserved. No write was submitted.")}
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
		diagnostics.AddError("Invalid service account recovery state", "Stored intent cannot be read. Independently reconcile before importing the verified GUID.")
		return nil, diagnostics
	}
	switch pending.Operation {
	case "create", "rename", "move", "update", "password", "delete":
		return &pending, diagnostics
	default:
		diagnostics.AddError("Invalid service account recovery operation", "Stored intent has an unknown operation. Independently reconcile before another write.")
		return nil, diagnostics
	}
}

// guardPending blocks later mutations until observation settles earlier intent or explicit verified-GUID recovery replaces it.
func guardPending(ctx context.Context, storage pendingData) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if pending != nil {
		diagnostics.AddError(
			"Service account mutation needs reconciliation",
			"Earlier intent remains unresolved. No write replay is allowed; verify the object and recover by GUID when necessary.",
		)
	}
	return diagnostics
}

// finishPending clears verified/acknowledged success and definite rejection without effects; all other outcomes keep the checkpoint.
func finishPending(
	ctx context.Context,
	storage pendingData,
	pending pendingMutation,
	model *ServiceAccountResourceModel,
	result shared.ObjectMutationResult,
	err error,
	success bool,
) diag.Diagnostics {
	if success || ((result.State == shared.ObjectWriteNotAttempted || result.State == shared.ObjectWriteRejected) && !result.PartialEffectsPossible && !shared.IsUncertain(err)) {
		return clearPending(ctx, storage)
	}
	if !model.ID.IsUnknown() && !model.ID.IsNull() {
		pending.GUID = model.ID.ValueString()
	}
	return savePending(ctx, storage, pending)
}

// reconcilePending proves observable intent only. Failed creation stays blocked because Terraform taints it;
// a password effect is never verified by reading a secret or attempting a login.
func reconcilePending(ctx context.Context, storage pendingData, observed *service_accounts.ServiceAccount) diag.Diagnostics {
	pending, diagnostics := loadPending(ctx, storage)
	if diagnostics.HasError() || pending == nil {
		return diagnostics
	}
	matched := observed != nil && observed.GUID != nil && observed.GUID.String() == pending.GUID && pending.GUID != ""
	if pending.Operation == "create" || pending.Operation == "password" || pending.Operation == "delete" {
		matched = false
	}
	if pending.Name != "" {
		matched = matched && observed != nil && observed.Name != nil && *observed.Name == pending.Name
	}
	if pending.ParentDN != "" {
		parent, valid := "", false
		if observed != nil {
			parent, valid = identity.ParentDN(observed.DistinguishedName)
		}
		matched = matched && valid && strings.EqualFold(parent, pending.ParentDN)
	}
	matched = matched && matchesPendingProperties(pending, observed)
	if matched {
		diagnostics.Append(clearPending(ctx, storage)...)
	} else {
		diagnostics.AddError(
			"Service account mutation still unresolved",
			"Readback did not prove the intended effect. State and intent remain; reconcile independently before another write.",
		)
	}
	return diagnostics
}

// matchesPendingProperties requires complete returned values; omission never proves an uncertain clear.
func matchesPendingProperties(pending *pendingMutation, observed *service_accounts.ServiceAccount) bool {
	if observed == nil {
		return false
	}
	for attribute, expected := range pending.Properties {
		if expected == nil {
			return false
		}
		value := lifecycle.ObserveString(observed.Raw, attribute)
		if !value.Usable || value.Diagnostics.HasError() || value.Value.ValueString() != *expected {
			return false
		}
	}
	ignored := ServiceAccountResourceModel{}
	control := controlObservation(observed.AccountControl)
	for _, field := range boolProperties(&ignored, &control) {
		if expected, selected := pending.Flags[field.name]; selected && (*field.target == nil || **field.target != expected) {
			return false
		}
	}
	if pending.Encryption != nil && (observed.SupportedEncryptionTypes == nil || uint32(*observed.SupportedEncryptionTypes)&0x1f != *pending.Encryption) {
		return false
	}
	if pending.Compound != nil && (observed.CompoundIdentitySupported == nil || *observed.CompoundIdentitySupported != *pending.Compound) {
		return false
	}
	if pending.SPNs != nil {
		if observed.ServicePrincipalNames.State != shared.AttributePresent {
			return false
		}
		actual := []string{}
		for _, raw := range observed.ServicePrincipalNames.Values {
			value, err := raw.Text()
			if err != nil {
				return false
			}
			actual = append(actual, value)
		}
		if !sameStringSet(actual, *pending.SPNs) {
			return false
		}
	}
	if pending.Readers != nil {
		if !observed.PasswordReaderDescriptorReturned {
			return false
		}
		actual := []string{}
		for _, sid := range observed.PasswordReaders {
			actual = append(actual, sid.String())
		}
		if !sameStringSet(actual, *pending.Readers) {
			return false
		}
	}
	return pending.Expiration == nil || observed.AccountExpires != nil && observed.AccountExpires.LDAPString() == strconv.FormatInt(*pending.Expiration, 10)
}

// pendingForUpdate captures only selected typed changes, including explicit clears, without storing secret material.
func pendingForUpdate(request service_accounts.UpdateServiceAccountRequest, model *ServiceAccountResourceModel) pendingMutation {
	pending := pendingMutation{Operation: "update", GUID: model.ID.ValueString(), Properties: map[string]*string{}, Flags: map[string]bool{}}
	for _, field := range textProperties(model, &request.Properties) {
		if *field.target != nil {
			pending.Properties[field.ldap] = *field.target
		}
	}
	for _, change := range request.Changes {
		if change.Kind == shared.AttributeClear {
			pending.Properties[change.Attribute] = nil
		}
	}
	for _, field := range boolProperties(model, &request.Control) {
		if *field.target != nil {
			pending.Flags[field.name] = **field.target
		}
	}
	if request.Properties.AccountExpires != nil {
		value := model.AccountExpires.ValueInt64()
		pending.Expiration = &value
	}
	if request.Properties.KerberosEncryptionTypes != nil {
		value := uint32(*request.Properties.KerberosEncryptionTypes)
		pending.Encryption = &value
	}
	pending.Compound = request.Properties.CompoundIdentitySupported
	pending.SPNs = request.Properties.ServicePrincipalNames
	if request.PasswordReaders != nil {
		values := make([]string, 0, len(*request.PasswordReaders))
		for _, reader := range *request.PasswordReaders {
			values = append(values, reader.String())
		}
		pending.Readers = &values
	}
	return pending
}

// sameStringSet compares complete observations without depending on LDAP multivalue ordering.
func sameStringSet(left, right []string) bool {
	a, b := slices.Clone(left), slices.Clone(right)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}
