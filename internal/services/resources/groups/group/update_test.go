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

func TestGroupUpdatePatchesOnlySelectedPropertiesAndFlags(t *testing.T) {
	current := creationModel()
	current.Description = types.StringValue("retain without ownership")
	plan := current
	plan.Description = types.StringNull()
	target, _ := shared.IdentityByGUID("b836f3d7-6ab2-45f4-a4db-67827ee3be44")
	request, changed, diagnostics := constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || changed || request.Category != nil || request.Scope != nil || request.Description != nil || len(request.Changes) != 0 {
		t.Fatal("omitted property or unchanged flags acquired ownership")
	}
	plan.Category = types.StringValue("distribution")
	request, changed, diagnostics = constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || !changed || request.Category == nil || *request.Category != groups.GroupDistribution || request.Scope != nil ||
		request.Description != nil || request.Options.ExpectedVersion != nil || len(request.Changes) != 0 {
		t.Fatal("category update replaced unrelated flags or properties")
	}
	plan = current
	plan.Scope = types.StringValue("universal")
	request, changed, diagnostics = constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || !changed || request.Scope == nil || *request.Scope != groups.GroupUniversal || request.Category != nil || len(request.Changes) != 0 {
		t.Fatal("scope conversion acquired category or raw flag ownership")
	}
	plan = current
	plan.Description = types.StringValue("")
	request, changed, diagnostics = constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || !changed || request.Description == nil || *request.Description != "" || len(request.Changes) != 0 {
		t.Fatal("literal empty property became an implicit clear")
	}
}

func TestGroupExplicitClearIsNotReplayedAfterAcknowledgement(t *testing.T) {
	current := creationModel()
	plan := current
	plan.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("description")})
	target, _ := shared.IdentityByGUID("b836f3d7-6ab2-45f4-a4db-67827ee3be44")
	request, changed, diagnostics := constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || !changed || len(request.Changes) != 1 || request.Changes[0].Kind != shared.AttributeClear ||
		request.Changes[0].Attribute != "description" || request.Description != nil {
		t.Fatal("new explicit clear intent was not preserved")
	}
	current = plan
	request, changed, diagnostics = constructUpdate(&current, &plan, target)
	if diagnostics.HasError() || changed || len(request.Changes) != 0 {
		t.Fatal("acknowledged explicit clear was replayed")
	}
}

func TestGroupUpdateRefusesInvalidLaterPhasesBeforeLocationChanges(t *testing.T) {
	current := creationModel()
	target, _ := shared.IdentityByGUID("b836f3d7-6ab2-45f4-a4db-67827ee3be44")
	for name, modify := range map[string]func(*GroupResourceModel){
		"unknown scope":    func(model *GroupResourceModel) { model.Scope = types.StringUnknown() },
		"invalid category": func(model *GroupResourceModel) { model.Category = types.StringValue("raw-mask") },
		"unknown manager":  func(model *GroupResourceModel) { model.ManagedByDN = types.StringUnknown() },
		"invalid manager":  func(model *GroupResourceModel) { model.ManagedByDN = types.StringValue("lookup-name") },
		"unknown clears":   func(model *GroupResourceModel) { model.ClearProperties = types.SetUnknown(types.StringType) },
		"membership clear": func(model *GroupResourceModel) {
			model.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("member")})
		},
		"raw flag clear": func(model *GroupResourceModel) {
			model.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("groupType")})
		},
		"conflicting property": func(model *GroupResourceModel) {
			model.Description = types.StringValue("configured")
			model.ClearProperties = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("description")})
		},
	} {
		t.Run(name, func(t *testing.T) {
			plan := current
			plan.Name = types.StringValue("renamed")
			plan.ParentDN = types.StringValue("OU=Other,DC=unit,DC=invalid")
			modify(&plan)
			request, changed, diagnostics := constructUpdate(&current, &plan, target)
			if !diagnostics.HasError() || changed || request.Identity.Form() != (shared.Identity{}).Form() {
				t.Fatal("invalid later phase escaped as a usable mutation")
			}
		})
	}
	dn, _ := shared.IdentityByDN("CN=mutable,DC=unit,DC=invalid")
	if _, changed, diagnostics := constructUpdate(&current, &current, dn); !diagnostics.HasError() || changed {
		t.Fatal("mutable distinguished name was accepted as lifecycle identity")
	}
}
