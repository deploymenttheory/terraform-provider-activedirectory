// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

// TestTerraformUnitAcceptedCreationCannotBeReplaced prevents tainted replacement after a server-accepted create with failed verification.
func TestTerraformUnitAcceptedCreationCannotBeReplaced(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config := unitConfiguration(t)
	directory := newUnitDirectory()
	service := directory.callbacks(t)
	create := service.Create
	service.Create = func(ctx context.Context, request shared.CreateObjectRequest) (shared.ObjectMutationResult, error) {
		result, err := create(ctx, request)
		if err != nil {
			return result, err
		}
		result.Verified = false
		return result, shared.ErrObjectReadback
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: config, ExpectError: regexp.MustCompile("Object mutation still unresolved")},
		{Config: config, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["create"] != 1 || directory.calls["delete"] != 0 {
		t.Fatal("accepted creation was repeated or automatically rolled back")
	}
}

// TestTerraformUnitUncertainClearCannotReplay refuses to infer clear success from a later omitted attribute.
func TestTerraformUnitUncertainClearCannotReplay(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	cleared := regexp.MustCompile(`description\s*=\s*\["Zmlyc3Q="\],?\s*`).ReplaceAllString(initial, "")
	cleared = strings.Replace(cleared, "\n  properties", "\n  clear_attributes = [\"description\"]\n  properties", 1)
	directory := newUnitDirectory()
	service := directory.callbacks(t)
	update := service.Update
	service.Update = func(ctx context.Context, request shared.UpdateObjectRequest) (shared.ObjectMutationResult, error) {
		result, _ := update(ctx, request)
		result.State, result.Verified = shared.ObjectWriteUncertain, false
		return result, shared.ErrObjectWriteUncertain
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: initial},
		{Config: cleared, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: cleared, ExpectError: regexp.MustCompile("Object mutation still unresolved")},
		{Config: initial, Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	if directory.calls["update"] != 1 || directory.calls["delete"] != 0 {
		t.Fatal("uncertain clear was replayed or converted to automatic destruction")
	}
}

// TestTerraformUnitAcceptedDNNormalizationKeepsIntent preserves requested bytes and actual canonical observations without a perpetual diff.
func TestTerraformUnitAcceptedDNNormalizationKeepsIntent(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	requested := base64.StdEncoding.EncodeToString([]byte("cn=Boss,ou=People,dc=unit,dc=invalid"))
	canonical := "CN=Boss,OU=People,DC=unit,DC=invalid"
	config := strings.NewReplacer("description", "manager", "Zmlyc3Q=", requested).Replace(unitConfiguration(t))
	directory := newUnitDirectory()
	service := directory.callbacks(t)
	create := service.Create
	service.Create = func(ctx context.Context, request shared.CreateObjectRequest) (shared.ObjectMutationResult, error) {
		result, err := create(ctx, request)
		if err != nil {
			return result, err
		}
		directory.mu.Lock()
		defer directory.mu.Unlock()
		directory.object.Attributes["manager"] = [][]byte{[]byte(canonical)}
		result = directory.mutation(request.Options.Properties)
		result.Verified = false
		return result, shared.ErrObjectUnverifiable
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "attributes.manager.*", requested),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.manager.*", base64.StdEncoding.EncodeToString([]byte(canonical))))},
		{Config: config, PlanOnly: true},
	}})
	if directory.calls["update"] != 0 {
		t.Fatal("accepted schema normalization triggered an automatic replacement write")
	}
}

// TestTerraformUnitAllPhasePreflight checks invalid raw intent before an otherwise valid rename can be submitted.
func TestTerraformUnitAllPhasePreflight(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	initial := unitConfiguration(t)
	invalid := strings.NewReplacer(`"fixture"`, `"renamed"`, "Zmlyc3Q=", "not canonical base64").Replace(initial)
	directory := newUnitDirectory()
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)), Steps: []resource.TestStep{
		{Config: initial},
		{Config: invalid, ExpectError: regexp.MustCompile("Invalid raw attribute intent")},
	}})
	if directory.calls["rename"] != 0 || directory.calls["update"] != 0 {
		t.Fatal("invalid later raw intent allowed an earlier mutation phase")
	}
}
