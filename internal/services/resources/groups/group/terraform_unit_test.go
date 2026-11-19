// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/groups"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func unitConfiguration(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// TestTerraformUnitGroupLifecycle verifies actual Framework planning/state/import rather than invoking CRUD methods directly.
func TestTerraformUnitGroupLifecycle(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	updated := strings.NewReplacer(`"initial"`, `"renamed"`, "OU=first,", "OU=second,", `"security"`, `"distribution"`, `"global"`, `"universal"`,
		"first description", "second description").Replace(initial)
	unmanaged := regexp.MustCompile(`(?m)^\s*description\s*=.*\n`).ReplaceAllString(updated, "")
	cleared := strings.Replace(unmanaged, "  scope", "  clear_properties = [\"description\"]\n  scope", 1)
	directory := newUnitDirectory(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "category", "security"),
			resource.TestCheckResourceAttr(ResourceName+".test", "scope", "global"),
		)},
		{Config: initial, PlanOnly: true},
		{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "name", "renamed"),
			resource.TestCheckResourceAttr(ResourceName+".test", "parent_dn", "OU=second,DC=unit,DC=invalid"),
			resource.TestCheckResourceAttr(ResourceName+".test", "category", "distribution"),
			resource.TestCheckResourceAttr(ResourceName+".test", "scope", "universal"),
		)},
		{Config: unmanaged, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
			}},
		},
		{Config: cleared, Check: resource.TestCheckNoResourceAttr(ResourceName+".test", "description")},
		{Config: cleared, PlanOnly: true},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for operation, expected := range map[string]int{"create": 1, "rename": 1, "move": 1, "update": 2, "delete": 1} {
		if directory.calls[operation] != expected {
			t.Errorf("%s calls=%d expected=%d", operation, directory.calls[operation], expected)
		}
	}
	if directory.flags&0x40000000 == 0 || directory.exists {
		t.Fatal("typed updates acquired raw flag ownership or destroy did not retire the group")
	}
}

func TestTerraformUnitAcceptedGroupCreationCannotBeReplaced(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	configuration := unitConfiguration(t)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	create := service.Create
	service.Create = func(ctx context.Context, request groups.CreateGroupRequest) (groups.GroupMutationResult, error) {
		result, err := create(ctx, request)
		if err != nil {
			return result, err
		}
		result.Mutation.Verified = false
		return result, shared.ErrObjectReadback
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: configuration, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: configuration, ExpectError: regexp.MustCompile("Group mutation still unresolved")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{Config: configuration, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["create"] != 1 || directory.calls["delete"] != 0 {
		t.Fatal("accepted creation was replaced or deleted without independent reconciliation")
	}
}

func TestTerraformUnitUncertainGroupRenameWaitsForReadProof(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	renamed := strings.Replace(initial, `"initial"`, `"renamed"`, 1)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	service.Rename = func(context.Context, shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.calls["rename"]++
		guid, _ := shared.ParseGUID(directory.GUID)
		return shared.ObjectMutationResult{State: shared.ObjectWriteUncertain, TargetGUID: &guid}, shared.ErrObjectWriteUncertain
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: initial},
		{Config: renamed, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: renamed, ExpectError: regexp.MustCompile("Group mutation still unresolved")},
		{Config: renamed, PreConfig: func() { directory.mu.Lock(); directory.Name = "renamed"; directory.mu.Unlock() }},
	}})
	if directory.calls["rename"] != 1 || directory.exists {
		t.Fatal("uncertain rename was replayed or read-proven identity was lost")
	}
}

func TestTerraformUnitRejectedScopeChangePreservesAcceptedRename(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	updated := strings.NewReplacer(`"initial"`, `"renamed"`, `"global"`, `"universal"`).Replace(initial)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	update := service.Update
	reject := true
	service.Update = func(ctx context.Context, request groups.UpdateGroupRequest) (groups.GroupMutationResult, error) {
		if !reject {
			return update(ctx, request)
		}
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.calls["update"]++
		guid, _ := shared.ParseGUID(directory.GUID)
		return groups.GroupMutationResult{Mutation: shared.ObjectMutationResult{State: shared.ObjectWriteRejected, TargetGUID: &guid}}, shared.ErrObjectConcurrency
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: initial},
		{Config: updated, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{Config: updated, PreConfig: func() { reject = false }, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
			resource.TestCheckResourceAttr(ResourceName+".test", "name", "renamed"),
			resource.TestCheckResourceAttr(ResourceName+".test", "scope", "universal"),
		)},
	}})
	if directory.calls["rename"] != 1 || directory.calls["update"] != 2 || directory.flags&0x40000000 == 0 {
		t.Fatal("a rejected concurrent scope change lost accepted rename, replayed automatically or acquired raw flag ownership")
	}
}

func TestTerraformUnitSDKVerifiedNoopIsAnObservationNotAWriteFailure(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	updated := strings.Replace(initial, `"global"`, `"universal"`, 1)
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	service.Update = func(_ context.Context, request groups.UpdateGroupRequest) (groups.GroupMutationResult, error) {
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.calls["update"]++
		// Another writer already established the desired semantic flags before the SDK's guarded preflight.
		directory.setProperties(request)
		mutation := directory.mutation()
		mutation.State = shared.ObjectWriteNotAttempted
		return groups.GroupMutationResult{Group: directory.snapshot(), Mutation: mutation}, nil
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: initial},
		{Config: updated, Check: resource.TestCheckResourceAttr(ResourceName+".test", "scope", "universal")},
		{Config: updated, PlanOnly: true},
	}})
	if directory.calls["update"] != 1 || directory.flags&0x40000000 == 0 {
		t.Fatal("verified no-op was retried or raw group flags were replaced")
	}
}
