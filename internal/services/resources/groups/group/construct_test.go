// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

func creationModel() GroupResourceModel {
	return GroupResourceModel{
		ClearProperties: types.SetNull(types.StringType),
		Name:            types.StringValue("Research, Europe"), SAMAccountName: types.StringValue("research-europe"),
		ParentDN: types.StringValue("OU=Groups,DC=unit,DC=invalid"), Category: types.StringValue("security"), Scope: types.StringValue("global"),
	}
}

func TestConstructGroupUsesSemanticFlagsAndExplicitParent(t *testing.T) {
	model := creationModel()
	model.Description = types.StringValue("")
	model.ManagedByDN = types.StringValue("CN=Manager,OU=People,DC=unit,DC=invalid")
	request, diagnostics := constructCreate(&model)
	if diagnostics.HasError() || request.Name != "Research, Europe" || request.Parent.Form() != shared.IdentityFormDN ||
		request.Category != groups.GroupSecurity || request.Scope != groups.GroupGlobal || request.SAMAccountName == nil ||
		*request.SAMAccountName != "research-europe" || request.Description == nil || *request.Description != "" || request.DisplayName != nil ||
		request.ManagedByDN == nil || len(request.Attributes) != 0 || len(request.Options.Properties) != 0 || request.Options.ExpectedVersion != nil {
		t.Fatal("typed identity, literal empty text or semantic group flag construction changed")
	}
	model.Category, model.Scope = types.StringValue("distribution"), types.StringValue("domain_local")
	request, diagnostics = constructCreate(&model)
	if diagnostics.HasError() || request.Category != groups.GroupDistribution || request.Scope != groups.GroupDomainLocal {
		t.Fatal("explicit distribution/domain-local mapping failed")
	}
	model.Scope = types.StringValue("universal")
	request, diagnostics = constructCreate(&model)
	if diagnostics.HasError() || request.Scope != groups.GroupUniversal {
		t.Fatal("universal mapping failed")
	}
}

func TestConstructGroupRefusesUnresolvedOrInvalidIntent(t *testing.T) {
	for name, modify := range map[string]func(*GroupResourceModel){
		"unknown name":     func(model *GroupResourceModel) { model.Name = types.StringUnknown() },
		"empty name":       func(model *GroupResourceModel) { model.Name = types.StringValue(" ") },
		"missing SAM":      func(model *GroupResourceModel) { model.SAMAccountName = types.StringNull() },
		"missing parent":   func(model *GroupResourceModel) { model.ParentDN = types.StringNull() },
		"invalid parent":   func(model *GroupResourceModel) { model.ParentDN = types.StringValue("Groups") },
		"unknown category": func(model *GroupResourceModel) { model.Category = types.StringUnknown() },
		"raw flag mask":    func(model *GroupResourceModel) { model.Category = types.StringValue("2147483650") },
		"missing scope":    func(model *GroupResourceModel) { model.Scope = types.StringNull() },
		"unknown scope":    func(model *GroupResourceModel) { model.Scope = types.StringUnknown() },
		"ambiguous scope":  func(model *GroupResourceModel) { model.Scope = types.StringValue("global,universal") },
		"unknown property": func(model *GroupResourceModel) { model.Description = types.StringUnknown() },
		"invalid text":     func(model *GroupResourceModel) { model.Description = types.StringValue(string([]byte{0xff})) },
		"NUL text":         func(model *GroupResourceModel) { model.DisplayName = types.StringValue("bad\x00value") },
		"manager lookup":   func(model *GroupResourceModel) { model.ManagedByDN = types.StringValue("manager@unit.invalid") },
		"clear at create": func(model *GroupResourceModel) {
			model.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("description")})
		},
	} {
		t.Run(name, func(t *testing.T) {
			model := creationModel()
			modify(&model)
			request, diagnostics := constructCreate(&model)
			if !diagnostics.HasError() || request.Name != "" || request.Parent.Form() != (shared.Identity{}).Form() || len(request.Attributes) != 0 {
				t.Fatal("invalid intent escaped construction as a usable group request")
			}
		})
	}
}
