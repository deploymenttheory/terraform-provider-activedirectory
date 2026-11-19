// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
)

// TestCreateFormsRefuseIncompatibleInputs checks complete preflight without constructing any password buffer on failure.
func TestCreateFormsRefuseIncompatibleInputs(t *testing.T) {
	for name, change := range map[string]func(*ServiceAccountResourceModel){
		"unknown kind":    func(m *ServiceAccountResourceModel) { m.Kind = types.StringUnknown() },
		"delegated class": func(m *ServiceAccountResourceModel) { m.Kind = types.StringValue("delegated") },
		"missing DNS":     func(m *ServiceAccountResourceModel) { m.DNSHostName = types.StringNull() },
		"reader name": func(m *ServiceAccountResourceModel) {
			m.PasswordReaderSIDs = types.SetValueMust(types.StringType, textElements([]string{"Administrators"}))
		},
		"unknown reader":    func(m *ServiceAccountResourceModel) { m.PasswordReaderSIDs = types.SetUnknown(types.StringType) },
		"oversize interval": func(m *ServiceAccountResourceModel) { m.ManagedPasswordIntervalDays = types.Int64Value(2147483648) },
		"outbound DNS":      func(m *ServiceAccountResourceModel) { m.OutboundAuthenticationOnly = types.BoolValue(true) },
		"standalone readers": func(m *ServiceAccountResourceModel) {
			m.Kind = types.StringValue("standalone")
			m.PasswordReaderSIDs = types.SetValueMust(types.StringType, nil)
		},
		"standalone interval": func(m *ServiceAccountResourceModel) {
			m.Kind = types.StringValue("standalone")
			m.ManagedPasswordIntervalDays = types.Int64Value(30)
		},
		"clear during create": func(m *ServiceAccountResourceModel) {
			m.ClearProperties = types.SetValueMust(types.StringType, textElements([]string{"description"}))
		},
	} {
		t.Run(name, func(t *testing.T) {
			model := creationModel()
			change(&model)
			request, diagnostics := constructCreate(&model, types.StringNull())
			if !diagnostics.HasError() || request.InitialPassword != nil {
				t.Fatal("invalid creation input survived preflight")
			}
		})
	}
	model := creationModel()
	model.DNSHostName = types.StringNull()
	model.OutboundAuthenticationOnly = types.BoolValue(true)
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.Kind != service_accounts.ServiceAccountGroupManaged || !request.OutboundAuthenticationOnly {
		t.Fatal("valid outbound gMSA form was rejected")
	}
}

// TestStandalonePasswordConstructionDelegatesRandomGenerationOrCopiesOnlyExplicitCreationInput verifies the two sMSA creation forms.
func TestStandalonePasswordConstructionDelegatesRandomGenerationOrCopiesOnlyExplicitCreationInput(t *testing.T) {
	model := creationModel()
	model.Kind = types.StringValue("standalone")
	model.DNSHostName = types.StringNull()
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.InitialPassword != nil {
		t.Fatal("provider should leave omitted sMSA password generation to the SDK")
	}
	model.PasswordWOVersion = types.Int64Value(1)
	request, diagnostics = constructCreate(&model, types.StringValue("unit-creation-only-secret"))
	defer clear(request.InitialPassword)
	if diagnostics.HasError() || string(request.InitialPassword) != "unit-creation-only-secret" {
		t.Fatal("explicit sMSA creation password was not copied into the SDK request")
	}
	model.Kind = types.StringValue("group_managed")
	model.DNSHostName = types.StringValue("managed.unit.invalid")
	if request, diagnostics := constructCreate(&model, types.StringValue("unit-creation-only-secret")); !diagnostics.HasError() || request.InitialPassword != nil {
		t.Fatal("gMSA accepted explicit password material")
	}
}
