// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

func (l *groupLab) groupConfig(t *testing.T, suffix, parent, category, scope, description string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(l.prefix))
	name, sam := l.prefix+"-"+suffix, fmt.Sprintf("tfg%x", digest[:6])
	return l.configuration(t, "resource.tf", "__NAME__", strconv.Quote(name), "__SAM__", strconv.Quote(sam), "__PARENT__", strconv.Quote(parent),
		"__CATEGORY__", strconv.Quote(category), "__SCOPE__", strconv.Quote(scope), "__DESCRIPTION_BLOCK__", description,
		"__FILTER__", strconv.Quote("(name="+name+")"))
}

// TestAccGroupLifecycle qualifies actual Terraform CRUD, conversion, import, generated configuration and all group discovery surfaces.
func TestAccGroupLifecycle(t *testing.T) {
	lab := newGroupLab(t)
	moveDN := "OU=" + lab.prefix + "-move," + lab.parentDN
	var moveGUID string
	createDestination := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		parent, _ := shared.IdentityByGUID(lab.parentGUID)
		protect := false
		description := "tfad-move:" + lab.target.RunID
		result, err := lab.observer.OrganizationalUnits.CreateOrganizationalUnit(t.Context(), organizational_units.CreateOrganizationalUnitRequest{
			Parent: parent, Name: lab.prefix + "-move", ProtectedFromAccidentalDeletion: &protect,
			Properties: organizational_units.OUProperties{DisplayName: &lab.marker, Description: &description},
		})
		if result.Object.TargetGUID != nil {
			moveGUID = result.Object.TargetGUID.String()
		}
		if err != nil || !result.Object.Verified || moveGUID == "" {
			t.Fatal("owned group move destination did not verify; preserve partial outcome")
		}
	}
	t.Cleanup(func() {
		if moveGUID == "" {
			return
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		target, _ := shared.IdentityByGUID(moveGUID)
		observed, err := lab.observer.OrganizationalUnits.OrganizationalUnit(context.Background(), target, shared.ResolveIdentityOptions{})
		if err != nil || !observed.Observation.Complete || observed.Unit == nil || observed.Unit.DisplayName == nil || *observed.Unit.DisplayName != lab.marker ||
			!strings.EqualFold(observed.Unit.DistinguishedName, moveDN) {
			t.Error("group move destination ownership did not verify")
			return
		}
		deleted, err := lab.observer.OrganizationalUnits.DeleteOrganizationalUnit(context.Background(), organizational_units.DeleteOrganizationalUnitRequest{Identity: target})
		if !lifecycle.ConfirmedDeletion(deleted, err) {
			t.Error("owned group move destination deletion did not verify")
		}
	})
	initial := lab.groupConfig(t, "initial", lab.parentDN, "security", "global", `description = "first description"`)
	updated := lab.groupConfig(t, "renamed", moveDN, "distribution", "universal", `description = "second description"`)
	cleared := lab.groupConfig(t, "renamed", moveDN, "distribution", "universal", `clear_properties = ["description"]`)
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"))
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "category", "security"), resource.TestCheckResourceAttr(ResourceName+".test", "scope", "global"))},
		{Config: initial, PlanOnly: true},
		{Config: updated, PreConfig: createDestination, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "category", "distribution"), resource.TestCheckResourceAttr(ResourceName+".test", "scope", "universal"),
			resource.TestCheckResourceAttr("data.activedirectory_groups.scoped", "groups.#", "1"))},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
			}},
		},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)}},
		{Config: cleared, Check: resource.ComposeAggregateTestCheckFunc(lab.capture, resource.TestCheckNoResourceAttr(ResourceName+".test", "description"))},
		{Config: cleared, PlanOnly: true},
	}})
}

// TestAccGroupMembersAndConstraints verifies that the server rejects a prohibited conversion and deleting a group leaves its member object alive.
func TestAccGroupMembersAndConstraints(t *testing.T) {
	lab := newGroupLab(t)
	initial := lab.groupConfig(t, "retained", lab.parentDN, "security", "universal", `description = "member preservation fixture"`)
	rejected := lab.groupConfig(t, "retained", lab.parentDN, "security", "global", `description = "member preservation fixture"`)
	var memberGUID, memberDN string
	seedMember := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil || lab.childGUID == "" {
			t.Fatal("managed group identity unavailable before member fixture creation")
		}
		parent, _ := shared.IdentityByGUID(lab.parentGUID)
		digest := sha256.Sum256([]byte(lab.prefix))
		sam := fmt.Sprintf("tfm%x", digest[:6])
		created, err := lab.observer.Groups.CreateGroup(t.Context(), groups.CreateGroupRequest{
			Parent: parent, Name: lab.prefix + "-member", Category: groups.GroupSecurity, Scope: groups.GroupUniversal,
			SAMAccountName: &sam, DisplayName: &lab.marker,
		})
		if created.Mutation.TargetGUID != nil {
			memberGUID = created.Mutation.TargetGUID.String()
		}
		memberDN = created.Mutation.TargetDN
		if err != nil || !created.Mutation.Verified || memberGUID == "" || memberDN == "" {
			t.Fatal("marker-owned universal member group did not verify")
		}
		container, _ := shared.IdentityByGUID(lab.childGUID)
		member, _ := shared.IdentityByGUID(memberGUID)
		linked, err := lab.observer.Memberships.AddGroupMembers(t.Context(), memberships.AddGroupMembersRequest{
			Identity: container, Members: []shared.Identity{member}, Mode: memberships.GroupMembershipStrict,
		})
		if err != nil || linked.Mutation.State != shared.ObjectWriteAccepted {
			t.Fatal("owned group membership addition was not acknowledged; no retry is authorized")
		}
		read, err := lab.observer.Directory.ResolveIdentity(t.Context(), container, shared.ResolveIdentityOptions{Properties: []string{"objectGUID", "member"}})
		if err != nil || !read.Complete || read.Object == nil {
			t.Fatal("owned member link readback incomplete")
		}
		value, valueErr := read.Object.Values("member").Single()
		text, textErr := value.Text()
		if valueErr != nil || textErr != nil || !strings.EqualFold(text, memberDN) {
			t.Fatal("owned universal member link did not independently verify")
		}
	}
	verifyMember := func() error {
		if memberGUID == "" {
			return errLabOwnership
		}
		target, _ := shared.IdentityByGUID(memberGUID)
		read, err := lab.observer.Groups.Group(context.Background(), target, shared.ResolveIdentityOptions{})
		if err != nil || !read.Observation.Complete || read.Group == nil || read.Group.GUID == nil || read.Group.GUID.String() != memberGUID ||
			read.Group.DisplayName == nil || *read.Group.DisplayName != lab.marker || !strings.EqualFold(read.Group.DistinguishedName, memberDN) ||
			read.Group.Scope == nil || *read.Group.Scope != groups.GroupUniversal {
			return errLabOwnership
		}
		return nil
	}
	t.Cleanup(func() {
		if memberGUID == "" {
			return
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if err := verifyMember(); err != nil {
			t.Error("owned group member identity changed before cleanup")
			return
		}
		target, _ := shared.IdentityByGUID(memberGUID)
		deleted, err := lab.observer.Groups.DeleteGroup(context.Background(), groups.DeleteGroupRequest{Identity: target})
		if !lifecycle.ConfirmedDeletion(deleted.Mutation, err) {
			t.Error("owned member group deletion did not independently verify")
		}
	})
	checkDestroy := func(state *terraform.State) error {
		if err := lab.checkDestroy(state); err != nil {
			return err
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		return verifyMember()
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: checkDestroy, Steps: []resource.TestStep{
		{Config: initial, Check: lab.capture},
		{Config: initial, PlanOnly: true, PreConfig: seedMember},
		{Config: rejected, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{Config: initial, Destroy: true},
	}})
}
