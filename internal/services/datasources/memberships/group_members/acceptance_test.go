// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
)

// TestAccGroupMembersDiscovery qualifies real direct/recursive queries over an owned cyclic graph, bounds, empty values and renamed endpoints.
func TestAccGroupMembersDiscovery(t *testing.T) {
	lab := newGroupMembersLab(t)
	root := lab.createNamedSubject(t, "group", "root")
	nested := lab.createNamedSubject(t, "group", "nested")
	deep := lab.createNamedSubject(t, "group", "deep")
	empty := lab.createNamedSubject(t, "group", "empty")
	first := lab.createNamedSubject(t, "user", "first")
	second := lab.createNamedSubject(t, "user", "second")
	computer := lab.createSubject(t, "computer")
	edges := [][2]labSubject{{root, first}, {root, nested}, {nested, first}, {nested, second}, {nested, deep}, {deep, root}, {deep, computer}}
	for _, edge := range edges {
		lab.setLink(t, edge[0], edge[1], true)
	}
	configuration := lab.configuration(t, "discovery.tf", "__GROUP_GUID__", root.GUID, "__EMPTY_GUID__", empty.GUID)
	proof := func(state *terraform.State) error {
		if err := lab.checkSecrets(state); err != nil {
			return err
		}
		for _, edge := range edges {
			if err := lab.assertLink(edge[0], edge[1], true); err != nil {
				return err
			}
		}
		for _, subject := range lab.subjects {
			if _, err := lab.ownedObject(context.Background(), subject); err != nil {
				return err
			}
		}
		if err := lab.memberSet(state, "direct", root.GUID, []labSubject{first, nested}); err != nil {
			return err
		}
		return lab.memberSet(state, "recursive", root.GUID, []labSubject{first, second, computer})
	}
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), Steps: []resource.TestStep{
		{Config: configuration, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(proof,
			resource.TestCheckResourceAttr("data."+DataSourceName+".direct", "complete", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".recursive", "complete", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".recursive", "groups_visited", "3"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".depth", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".depth", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".objects", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".objects", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".groups", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".groups", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".ranges", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".ranges", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".bytes", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".bytes", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".empty", "members.#", "0"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".empty", "complete", "false"),
		)},
		{Config: configuration, PlanOnly: true},
		{Config: configuration, PreConfig: func() {
			lab.renameSubject(t, root)
			lab.renameSubject(t, first)
		}, ConfigPlanChecks: checks, Check: proof},
		{Config: configuration, PlanOnly: true},
	}})
	if err := proof(nil); err != nil {
		t.Fatal("read-only discovery changed an owned endpoint or link")
	}
}

// TestAccGroupMembersPrimary observes the fixture user's unchanged built-in primary group without modifying or owning that group.
func TestAccGroupMembersPrimary(t *testing.T) {
	lab := newGroupMembersLab(t)
	user := lab.createNamedSubject(t, "user", "primary")
	groupGUID := lab.primaryGroup(t, user)
	target := labSubject{GUID: groupGUID, Kind: "group"}
	raw, err := lab.observer.Memberships.GroupMemberLinks(t.Context(), lab.identity(target), memberships.GroupMemberLinksOptions{MaxValues: 20})
	if err != nil || !raw.Complete || len(raw.Members) != 0 {
		t.Fatal("readonly primary-group control is not the expected direct-attribute-empty fixture; no built-in mutation is permitted")
	}
	configuration := lab.configuration(t, "primary.tf", "__GROUP_GUID__", groupGUID)
	proof := func(state *terraform.State) error {
		if err := lab.checkSecrets(state); err != nil {
			return err
		}
		if state == nil {
			return nil
		}
		observed := state.RootModule().Resources["data."+DataSourceName+".primary"]
		if observed == nil || observed.Primary == nil || observed.Primary.ID != groupGUID {
			return errLabOwnership
		}
		count, err := strconv.Atoi(observed.Primary.Attributes["members.#"])
		if err != nil || count < 1 {
			return errLabOwnership
		}
		for index := range count {
			prefix := fmt.Sprintf("members.%d.", index)
			if observed.Primary.Attributes[prefix+"id"] == user.GUID && observed.Primary.Attributes[prefix+"primary"] == "true" {
				return nil
			}
		}
		return errLabOwnership
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), Steps: []resource.TestStep{
		{Config: configuration, Check: resource.ComposeAggregateTestCheckFunc(proof,
			resource.TestCheckResourceAttr("data."+DataSourceName+".primary", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".primary", "visibility_known", "false"),
		)},
		{Config: configuration, PlanOnly: true},
	}})
}

// checkSecrets scans the complete Terraform state rather than only selected directory output fields.
func (l *groupMembersLab) checkSecrets(state *terraform.State) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return errLabOwnership
	}
	for _, secret := range l.secrets {
		if secret != "" && strings.Contains(string(encoded), secret) {
			return errLabOwnership
		}
	}
	return nil
}

// memberSet verifies exact fixture identities and GUID ordering without relying on names as membership identity.
func (l *groupMembersLab) memberSet(state *terraform.State, name, group string, subjects []labSubject) error {
	if state == nil {
		return nil
	}
	observed := state.RootModule().Resources["data."+DataSourceName+"."+name]
	if observed == nil || observed.Primary == nil || observed.Primary.ID != group || observed.Primary.Attributes["members.#"] != strconv.Itoa(len(subjects)) {
		return errLabOwnership
	}
	wanted := map[string]bool{}
	for _, subject := range subjects {
		wanted[subject.GUID] = true
	}
	previous := ""
	for index := range len(subjects) {
		prefix := fmt.Sprintf("members.%d.", index)
		id := observed.Primary.Attributes[prefix+"id"]
		if !wanted[id] || (index > 0 && id <= previous) || observed.Primary.Attributes[prefix+"sid"] == "" || observed.Primary.Attributes[prefix+"distinguished_name"] == "" {
			return errLabOwnership
		}
		if !strings.EqualFold(observed.Primary.Attributes[prefix+"source_host"], l.target.Host) {
			return errLabOwnership
		}
		delete(wanted, id)
		previous = id
	}
	return nil
}
