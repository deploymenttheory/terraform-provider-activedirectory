// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccGroupMemberDeniedAndHidden verifies ordinary-actor refusal and distinguishes hidden direct values from an empty group.
func TestAccGroupMemberDeniedAndHidden(t *testing.T) {
	lab := newGroupMemberLab(t)
	group := lab.createNamedSubject(t, "group", "denied")
	member := lab.createSubject(t, "user")
	unmanaged := lab.createNamedSubject(t, "group", "unmanaged")
	lab.setLink(t, group, unmanaged, true)
	hidden, hiddenMember, hiddenReadable := lab.hiddenFixture(t)
	configuration := func(attempt string) string {
		return lab.configuration(t, "denied.tf", "__GROUP_GUID__", group.GUID, "__MEMBER_GUID__", member.GUID,
			"__UNMANAGED_GUID__", unmanaged.GUID, "# __ATTEMPT_BLOCK__", "count = "+attempt)
	}
	baseline := configuration("0")
	unchanged := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		if err := lab.assertLink(group, member, false); err != nil {
			return err
		}
		if err := lab.assertLink(group, unmanaged, true); err != nil {
			return err
		}
		return lab.assertLink(hidden, hiddenMember, true)
	}
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	hiddenLookup := baseline + lab.configuration(t, "hidden.tf", "__GROUP_GUID__", hidden.GUID, "__MEMBER_GUID__", hiddenMember.GUID)
	hiddenStep := resource.TestStep{Config: hiddenLookup}
	if hiddenReadable {
		hiddenStep.Check = resource.TestCheckResourceAttr("data.activedirectory_group_member.hidden", "id", hidden.GUID+"/"+hiddenMember.GUID)
	} else {
		hiddenStep.ExpectError = regexp.MustCompile("Direct membership unavailable")
	}
	hiddenCollection := baseline + lab.configuration(t, "hidden-collection.tf", "__GROUP_GUID__", hidden.GUID)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: baseline, ConfigPlanChecks: checks, Check: unchanged},
		{Config: configuration("1"), ConfigPlanChecks: checks, ExpectError: regexp.MustCompile("Membership mutation rejected")},
		{Config: baseline, Check: unchanged},
		hiddenStep,
		{Config: hiddenCollection, ExpectError: regexp.MustCompile("Membership discovery incomplete")},
		{Config: baseline, Check: unchanged},
		{Config: baseline, PlanOnly: true},
	}})
	if err := unchanged(nil); err != nil {
		t.Fatal("refused or hidden membership operations changed native directory links")
	}
}
