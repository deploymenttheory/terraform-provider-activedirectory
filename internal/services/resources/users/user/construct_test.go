// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func TestCreationRequiresExplicitNamesAndKeepsDisabledDefault(t *testing.T) {
	model := creationModel()
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.Parent.Form() != shared.IdentityFormDN || request.Class != "user" || request.InitialPassword != nil ||
		request.Properties.Enabled != nil || request.Properties.PasswordNotRequired != nil || request.Options.ExpectedVersion != nil {
		t.Fatal("creation guessed a parent, weakened account defaults, added an assertion or invented password intent")
	}
}

func TestCreationPasswordIsExplicitVersionedAndRedacted(t *testing.T) {
	const secret = "synthetic-initial-password"
	model := creationModel()
	model.PasswordWO = types.StringValue(secret)
	model.PasswordWOVersion = types.Int64Value(1)
	model.Enabled = types.BoolValue(true)
	request, diagnostics := constructCreate(&model, model.PasswordWO)
	defer clear(request.InitialPassword)
	if diagnostics.HasError() || string(request.InitialPassword) != secret || request.Properties.Enabled == nil || !*request.Properties.Enabled {
		t.Fatal("explicit password or requested staged enablement was lost")
	}
	if strings.Contains(fmt.Sprintf("%v %#v %v %#v", model, model, request, request), secret) {
		t.Fatal("model or SDK request formatting disclosed write-only input")
	}
	model.PasswordWOVersion = types.Int64Value(0)
	rejected, diagnostics := constructCreate(&model, model.PasswordWO)
	if !diagnostics.HasError() || rejected.InitialPassword != nil {
		t.Fatal("invalid version allocated a password request")
	}
}

func TestUnknownCreationInputsAllocateNoSecretBuffer(t *testing.T) {
	for _, field := range []string{"name", "sam", "parent", "class", "description", "enabled", "expiration", "version", "clear", "password"} {
		t.Run(field, func(t *testing.T) {
			model := creationModel()
			model.PasswordWOVersion = types.Int64Value(1)
			password := types.StringValue("synthetic-secret")
			switch field {
			case "name":
				model.Name = types.StringUnknown()
			case "sam":
				model.SAMAccountName = types.StringUnknown()
			case "parent":
				model.ParentDN = types.StringUnknown()
			case "class":
				model.ObjectClass = types.StringUnknown()
			case "description":
				model.Description = types.StringUnknown()
			case "enabled":
				model.Enabled = types.BoolUnknown()
			case "expiration":
				model.AccountExpires = types.Int64Unknown()
			case "version":
				model.PasswordWOVersion = types.Int64Unknown()
			case "clear":
				model.ClearProperties = types.SetUnknown(types.StringType)
			case "password":
				password = types.StringUnknown()
			}
			request, diagnostics := constructCreate(&model, password)
			if !diagnostics.HasError() || request.InitialPassword != nil {
				t.Fatal("unresolved creation input reached a secret-bearing request")
			}
		})
	}
}

func TestCreationDoesNotInferPasswordNotRequired(t *testing.T) {
	model := creationModel()
	model.Enabled = types.BoolValue(true)
	if _, diagnostics := constructCreate(&model, types.StringNull()); !diagnostics.HasError() {
		t.Fatal("enablement without password silently weakened account requirements")
	}
	model.PasswordNotRequired = types.BoolValue(true)
	request, diagnostics := constructCreate(&model, types.StringNull())
	if diagnostics.HasError() || request.Properties.PasswordNotRequired == nil || !*request.Properties.PasswordNotRequired || request.InitialPassword != nil {
		t.Fatal("explicit SDK-supported password-not-required intent was not preserved")
	}
}

func TestPropertiesKeepOmissionEmptyTextAndExpirationSentinelsDistinct(t *testing.T) {
	model := creationModel()
	model.Description = types.StringValue("")
	for _, stamp := range []int64{0, math.MaxInt64} {
		model.AccountExpires = types.Int64Value(stamp)
		properties, diagnostics := constructProperties(&model)
		if diagnostics.HasError() || properties.Description == nil || *properties.Description != "" || properties.Department != nil ||
			properties.AccountExpires == nil || *properties.AccountExpires != shared.FileTime(stamp) {
			t.Fatal("property construction conflated omission, empty text or FILETIME sentinel values")
		}
	}
}

func creationModel() UserResourceModel {
	return UserResourceModel{
		Name: types.StringValue("fixture"), SAMAccountName: types.StringValue("fixture"), ParentDN: types.StringValue("OU=fixtures,DC=unit,DC=invalid"),
		ObjectClass: types.StringNull(), PasswordWO: types.StringNull(), PasswordWOVersion: types.Int64Value(0), ClearProperties: types.SetNull(types.StringType),
	}
}
