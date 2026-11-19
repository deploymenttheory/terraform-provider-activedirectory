// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestAccGroupMemberLifecycle qualifies actual direct links, discovery, imports, drift and replacement without owning either endpoint.
func TestAccGroupMemberLifecycle(t *testing.T) {
	lab := newGroupMemberLab(t)
	group := lab.createNamedSubject(t, "group", "owner")
	memberGroup := lab.createNamedSubject(t, "group", "directgroup")
	nested := lab.createNamedSubject(t, "group", "nested")
	empty := lab.createNamedSubject(t, "group", "empty")
	user := lab.createNamedSubject(t, "user", "user")
	alternate := lab.createNamedSubject(t, "user", "alternate")
	indirect := lab.createNamedSubject(t, "user", "indirect")
	computer := lab.createNamedSubject(t, "computer", "computer")
	lab.setLink(t, nested, indirect, true)
	lab.setLink(t, group, nested, true)
	lab.setLink(t, group, user, true)
	initial := lab.configuration(t, "resource.tf",
		"__GROUP_GUID__", group.GUID, "__MEMBER_GROUP_GUID__", memberGroup.GUID, "__USER_GUID__", user.GUID,
		"__COMPUTER_GUID__", computer.GUID, "__EMPTY_GROUP_GUID__", empty.GUID,
	)
	replaced := strings.Replace(initial, user.GUID, alternate.GUID, 1)
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__GROUP_GUID__", group.GUID)
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	proof := func(selected labSubject) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			if err := lab.capture(state); err != nil {
				return err
			}
			for _, member := range []labSubject{selected, memberGroup, computer, nested} {
				if err := lab.assertLink(group, member, true); err != nil {
					return err
				}
			}
			if err := lab.assertLink(group, indirect, false); err != nil {
				return err
			}
			return lab.assertLink(nested, indirect, true)
		}
	}
	destroyProof := func(state *terraform.State) error {
		if err := lab.checkDestroy(state); err != nil {
			return err
		}
		for _, member := range []labSubject{user, alternate, memberGroup, computer} {
			if err := lab.assertLink(group, member, false); err != nil {
				return err
			}
		}
		if err := lab.assertLink(group, nested, true); err != nil {
			return err
		}
		return lab.assertLink(nested, indirect, true)
	}
	primaryGUID := lab.primaryGroup(t, user)
	nestedLookup := initial + lab.configuration(t, "not-direct.tf", "__GROUP_GUID__", group.GUID, "__MEMBER_GUID__", indirect.GUID)
	primaryLookup := initial + lab.configuration(t, "not-direct.tf", "__GROUP_GUID__", primaryGUID, "__MEMBER_GUID__", user.GUID)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: destroyProof, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(proof(user),
			resource.TestCheckResourceAttr(ResourceName+".user", "id", group.GUID+"/"+user.GUID),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.owned", "members.#", "4"),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.owned", "complete", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.empty", "members.#", "0"),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.empty", "complete", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.bounded", "complete", "false"),
			resource.TestCheckResourceAttr("data.activedirectory_group_member_relationships.bounded", "limit_reached", "true"),
		)},
		{Config: initial, PlanOnly: true},
		{ResourceName: ResourceName + ".user", ImportState: true, ImportStateVerify: true},
		{ResourceName: ResourceName + ".computer", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".group", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)}},
		},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 4)}},
		{Config: nestedLookup, ExpectError: regexp.MustCompile("Direct membership unavailable")},
		{Config: primaryLookup, ExpectError: regexp.MustCompile("Direct membership unavailable")},
		{Config: initial, ConfigPlanChecks: checks, PreConfig: func() {
			lab.setLink(t, group, user, false)
			lab.renameSubject(t, user)
			lab.renameSubject(t, group)
		}, Check: proof(user)},
		{Config: replaced, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			labSecretPlan{secrets: lab.secrets}, plancheck.ExpectResourceAction(ResourceName+".user", plancheck.ResourceActionDestroyBeforeCreate),
		}}, Check: resource.ComposeAggregateTestCheckFunc(proof(alternate), resource.TestCheckResourceAttr(ResourceName+".user", "id", group.GUID+"/"+alternate.GUID))},
		{Config: replaced, PlanOnly: true},
	}})
	if err := destroyProof(nil); err != nil {
		t.Fatal("direct-link deletion changed unrelated membership or an independently owned endpoint")
	}
}

// primaryGroup resolves the fixture user's existing primary group read-only; no built-in group or membership is changed.
func (l *groupMemberLab) primaryGroup(t *testing.T, user labSubject) string {
	t.Helper()
	object, err := l.observer.Directory.ResolveIdentity(t.Context(), l.identity(user), shared.ResolveIdentityOptions{Properties: []string{"objectSid", "primaryGroupID"}})
	if err != nil || !object.Complete || object.Object == nil {
		t.Fatal("fixture primary-group metadata is unavailable")
	}
	value, err := object.Object.Values("objectSid").Single()
	sid, sidErr := value.SID()
	primary, primaryErr := object.Object.Values("primaryGroupID").Single()
	rid, textErr := primary.Text()
	if err != nil || sidErr != nil || primaryErr != nil || textErr != nil || rid != "513" {
		t.Fatal("fresh fixture user does not have the expected unchanged primary group")
	}
	parts := strings.Split(sid.String(), "-")
	primarySID := strings.Join(parts[:len(parts)-1], "-") + "-" + rid
	target, err := shared.IdentityBySID(primarySID)
	if err != nil {
		t.Fatal("fixture primary-group SID is invalid")
	}
	group, err := l.observer.Groups.Group(context.Background(), target, shared.ResolveIdentityOptions{})
	if err != nil || !group.Observation.Complete || group.Group == nil || group.Group.GUID == nil || group.Group.SID == nil || group.Group.SID.String() != primarySID {
		t.Fatal("fixture primary group could not be independently resolved")
	}
	return group.Group.GUID.String()
}
