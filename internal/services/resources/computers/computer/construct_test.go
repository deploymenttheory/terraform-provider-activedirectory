// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func creationModel() ComputerResourceModel {
	return ComputerResourceModel{
		Name: types.StringValue("TFManagedComputer"), SAMAccountName: types.StringValue("TFManagedComputer$"),
		ParentDN:        types.StringValue("OU=Computers,DC=unit,DC=invalid"),
		ClearProperties: types.SetNull(types.StringType), KerberosEncryptionTypes: types.SetNull(types.StringType),
	}
}

func TestComputerCreationKeepsDirectoryOnlyIdentityAndDisabledDefaults(t *testing.T) {
	model := creationModel()
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.Name != "TFManagedComputer" || request.Parent.Form() != shared.IdentityFormDN || request.Properties.SAMAccountName == nil ||
		*request.Properties.SAMAccountName != "TFManagedComputer$" || request.InitialPassword != nil || request.Control.Enabled != nil ||
		request.Control.PasswordNotRequired != nil || request.Options.ExpectedVersion != nil || len(request.Attributes) != 0 {
		t.Fatal("computer construction guessed identity, enabled an account, weakened password defaults or acquired raw ownership")
	}
}

func TestComputerPropertiesSeparateEncryptionBitsAndExpirationSentinels(t *testing.T) {
	model := creationModel()
	model.KerberosEncryptionTypes = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("aes128"), types.StringValue("aes256")})
	model.CompoundIdentitySupported = types.BoolValue(true)
	model.Description = types.StringValue("")
	model.AccountNotDelegated = types.BoolValue(true)
	for _, stamp := range []int64{0, math.MaxInt64} {
		model.AccountExpires = types.Int64Value(stamp)
		properties, control, diagnostics := constructProperties(&model)
		if diagnostics.HasError() || properties.KerberosEncryptionTypes == nil || *properties.KerberosEncryptionTypes != 24 ||
			properties.CompoundIdentitySupported == nil || !*properties.CompoundIdentitySupported || properties.AccountExpires == nil ||
			*properties.AccountExpires != shared.FileTime(stamp) || properties.Description == nil || *properties.Description != "" ||
			control.AccountNotDelegated == nil || !*control.AccountNotDelegated || control.Enabled != nil {
			t.Fatal("computer construction conflated omission, explicit empty text, expiration or individually selected account/encryption flags")
		}
	}
}

func TestComputerPasswordRequiresExplicitVersionAndRedactedFormatting(t *testing.T) {
	model := creationModel()
	model.Enabled = types.BoolValue(true)
	if _, diagnostics := constructCreate(&model, types.StringNull()); !diagnostics.HasError() {
		t.Fatal("new computer was enabled without password setup or explicit exemption")
	}
	model.PasswordWO, model.PasswordWOVersion = types.StringValue("synthetic-computer-secret"), types.Int64Value(1)
	request, diagnostics := constructCreate(&model, model.PasswordWO)
	defer clear(request.InitialPassword)
	if diagnostics.HasError() || string(request.InitialPassword) != model.PasswordWO.ValueString() || request.Control.Enabled == nil || !*request.Control.Enabled {
		t.Fatal("explicit initial computer password or enablement was lost")
	}
	if strings.Contains(fmt.Sprintf("%v %#v %v %#v", model, model, request, request), model.PasswordWO.ValueString()) {
		t.Fatal("computer model or SDK request formatting exposed a write-only password")
	}
}

func TestComputerCreationKeepsLogonPasswordIntentForPostCreationPhase(t *testing.T) {
	model := creationModel()
	model.ChangePasswordAtLogon = types.BoolValue(true)
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.Control.ChangePasswordAtLogon != nil || !model.ChangePasswordAtLogon.ValueBool() {
		t.Fatal("creation forwarded unsupported pwdLastSet semantics or discarded the separately requested post-creation intent")
	}
	_, control, diagnostics := constructProperties(&model)
	if diagnostics.HasError() || control.ChangePasswordAtLogon == nil || !*control.ChangePasswordAtLogon {
		t.Fatal("typed update construction lost the explicitly selected logon-password setting")
	}
}

func TestInvalidComputerIntentAllocatesNoPasswordRequest(t *testing.T) {
	for name, modify := range map[string]func(*ComputerResourceModel){
		"implicit SAM suffix": func(model *ComputerResourceModel) { model.SAMAccountName = types.StringValue("host") },
		"double SAM suffix":   func(model *ComputerResourceModel) { model.SAMAccountName = types.StringValue("host$$") },
		"empty account name":  func(model *ComputerResourceModel) { model.SAMAccountName = types.StringValue("$") },
		"unknown name":        func(model *ComputerResourceModel) { model.Name = types.StringUnknown() },
		"unknown parent":      func(model *ComputerResourceModel) { model.ParentDN = types.StringUnknown() },
		"unknown OS metadata": func(model *ComputerResourceModel) { model.OperatingSystem = types.StringUnknown() },
		"invalid manager":     func(model *ComputerResourceModel) { model.ManagedByDN = types.StringValue("lookup-principal") },
		"unknown control":     func(model *ComputerResourceModel) { model.TrustedForDelegation = types.BoolUnknown() },
		"unknown compound":    func(model *ComputerResourceModel) { model.CompoundIdentitySupported = types.BoolUnknown() },
		"unknown encryption":  func(model *ComputerResourceModel) { model.KerberosEncryptionTypes = types.SetUnknown(types.StringType) },
		"raw encryption mask": func(model *ComputerResourceModel) {
			model.KerberosEncryptionTypes = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("2147483648")})
		},
		"negative expiration": func(model *ComputerResourceModel) { model.AccountExpires = types.Int64Value(-1) },
		"missing version":     func(model *ComputerResourceModel) { model.PasswordWOVersion = types.Int64Null() },
		"contradictory settings": func(model *ComputerResourceModel) {
			model.PasswordNeverExpires, model.ChangePasswordAtLogon = types.BoolValue(true), types.BoolValue(true)
		},
	} {
		t.Run(name, func(t *testing.T) {
			model := creationModel()
			model.PasswordWOVersion = types.Int64Value(1)
			modify(&model)
			request, diagnostics := constructCreate(&model, types.StringValue("synthetic-computer-secret"))
			if !diagnostics.HasError() || request.InitialPassword != nil {
				t.Fatal("invalid computer configuration reached secret-bearing request construction")
			}
		})
	}
}
