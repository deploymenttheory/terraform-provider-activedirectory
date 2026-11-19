// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit/mocks"
)

// TestTerraformUnitLifecycle verifies the actual Framework/Terraform state contract over a mocked SDK boundary.
func TestTerraformUnitLifecycle(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	configBytes, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := string(configBytes)
	updated := strings.NewReplacer("\"initial\"", "\"renamed\"", "first description", "second description",
		"parent_dn                          = \"DC=unit,DC=invalid\"", "parent_dn                          = \"OU=second,DC=unit,DC=invalid\"").Replace(initial)
	unmanaged := strings.Replace(updated, "  description                        = \"second description\"\n", "", 1)
	cleared := strings.Replace(unmanaged, "  protected_from_accidental_deletion", "  clear_properties = [\"description\"]\n  protected_from_accidental_deletion", 1)
	directory := newUnitDirectory(t)
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: unitProviderFactories(t, directory.callbacks(t)),
		Steps: []resource.TestStep{
			{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
				resource.TestCheckResourceAttr(ResourceName+".test", "description", "first description"))},
			{Config: initial, PlanOnly: true},
			{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(ResourceName+".test", "id", fixtureGUID),
				resource.TestCheckResourceAttr(ResourceName+".test", "name", "renamed"),
				resource.TestCheckResourceAttr(ResourceName+".test", "parent_dn", "OU=second,DC=unit,DC=invalid"),
				resource.TestCheckResourceAttr(ResourceName+".test", "description", "second description"))},
			{Config: unmanaged, PlanOnly: true},
			{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
			{
				ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
				ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
					plancheck.ExpectKnownValue(ResourceName+".generated", tfjsonpath.New("id"), knownvalue.StringExact(fixtureGUID)),
					plancheck.ExpectKnownValue(ResourceName+".generated", tfjsonpath.New("description"), knownvalue.StringExact("second description")),
				}},
			},
			{Config: unmanaged, PlanOnly: true},
			{Config: cleared, Check: resource.TestCheckNoResourceAttr(ResourceName+".test", "description")},
			{Config: cleared, PlanOnly: true},
		},
	})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	for _, operation := range []string{"create", "rename", "move", "update", "delete"} {
		expected := 1
		if operation == "update" {
			expected = 2
		}
		if directory.calls[operation] != expected {
			t.Errorf("%s calls = %d; expected %d", operation, directory.calls[operation], expected)
		}
	}
	if directory.exists {
		t.Error("Terraform destroy did not remove the mock object")
	}
}

func TestTerraformUnitAcceptedCreateRetainsGUIDForExplicitImport(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	directory := newUnitDirectory(t)
	service := directory.callbacks(t)
	create := service.Create
	service.Create = func(ctx context.Context, request organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error) {
		result, err := create(ctx, request)
		if err != nil {
			return result, err
		}
		result.Object.Verified = false
		return result, shared.ErrObjectReadback
	}
	t.Chdir(t.TempDir())
	resource.Test(t, resource.TestCase{IsUnitTest: true, ProtoV6ProviderFactories: unitProviderFactories(t, service), Steps: []resource.TestStep{
		{Config: string(config), ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		// An operator retires this synthetic fixture after verifying the import. The final
		// refresh confirms absence; the provider must never replace the tainted creation.
		{Config: string(config), Destroy: true, PreConfig: func() { directory.mu.Lock(); directory.exists = false; directory.mu.Unlock() }},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["create"] != 1 || directory.exists {
		t.Error("failed apply lost the accepted creation or replayed Add instead of reconciling its GUID")
	}
}

func TestTerraformUnitUncertainRenameCannotReplayBeforeReconciliation(t *testing.T) {
	acceptance.IsolateUnitEnvironment(t)
	config, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	initial := string(config)
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
		{Config: renamed, ExpectError: regexp.MustCompile("OU mutation still unresolved")},
		{Config: renamed, PreConfig: func() { directory.mu.Lock(); directory.Name = "renamed"; directory.mu.Unlock() }},
	}})
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.calls["rename"] != 1 || directory.exists {
		t.Error("uncertain rename was replayed or independently settled state was discarded")
	}
}

type unitDirectory struct {
	mu                 sync.Mutex
	GUID               string `json:"guid"`
	Name               string `json:"name"`
	ParentDN           string `json:"parentDn"`
	Description        string `json:"description"`
	protect            bool
	descriptionPresent bool
	exists             bool
	calls              map[string]int
}

func newUnitDirectory(t *testing.T) *unitDirectory {
	t.Helper()
	payload, err := os.ReadFile("tests/responses/organizational_unit.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := &unitDirectory{calls: map[string]int{}, descriptionPresent: true}
	if err := json.Unmarshal(payload, directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func (d *unitDirectory) snapshot() *organizational_units.OrganizationalUnit {
	guid, _ := shared.ParseGUID(d.GUID)
	name, protect := d.Name, d.protect
	dn := "OU=" + d.Name + "," + d.ParentDN
	unit := &organizational_units.OrganizationalUnit{
		GUID: &guid, Name: &name, DistinguishedName: dn, ProtectedFromAccidentalDeletion: &protect,
		Raw: shared.Object{DistinguishedName: dn, RequestedProperties: []string{"description"}, Attributes: map[string][][]byte{
			"objectguid": {guid.Bytes()}, "name": {[]byte(name)},
		}},
	}
	if d.descriptionPresent {
		unit.Raw.Attributes["description"] = [][]byte{[]byte(d.Description)}
	}
	return unit
}

func (d *unitDirectory) mutation() shared.ObjectMutationResult {
	unit := d.snapshot()
	return shared.ObjectMutationResult{
		State: shared.ObjectWriteAccepted, TargetGUID: unit.GUID, TargetDN: unit.DistinguishedName, After: &unit.Raw,
		Verified: true, ReadbackComplete: true,
	}
}

func (d *unitDirectory) callbacks(t *testing.T) *featuremocks.Service {
	t.Helper()
	checkTarget := func(id shared.Identity, options shared.ObjectMutationOptions) {
		if id.Form() != shared.IdentityFormGUID || id.String() != d.GUID || options.ExpectedVersion != nil {
			t.Error("resource lost GUID addressing or introduced an assertion control")
		}
	}
	return &featuremocks.Service{
		Read: func(_ context.Context, id shared.Identity, _ shared.ResolveIdentityOptions) (organizational_units.OrganizationalUnitResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(id, shared.ObjectMutationOptions{})
			if !d.exists {
				return organizational_units.OrganizationalUnitResult{}, shared.ErrDirectoryObjectNotFound
			}
			unit := d.snapshot()
			return organizational_units.OrganizationalUnitResult{Unit: unit, Observation: shared.IdentityResult{Complete: true, Object: &unit.Raw}}, nil
		},
		Create: func(_ context.Context, req organizational_units.CreateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return organizational_units.OrganizationalUnitMutationResult{}, shared.ErrObjectAlreadyExists
			}
			if req.Parent.Form() != shared.IdentityFormDN || req.Options.ExpectedVersion != nil {
				t.Error("invalid create parent or assertion")
			}
			d.Name, d.ParentDN, d.exists = req.Name, req.Parent.String(), true
			if req.Properties.Description != nil {
				d.Description = *req.Properties.Description
				d.descriptionPresent = true
			}
			if req.ProtectedFromAccidentalDeletion != nil {
				d.protect = *req.ProtectedFromAccidentalDeletion
			}
			unit := d.snapshot()
			return organizational_units.OrganizationalUnitMutationResult{
				Object: d.mutation(), Unit: unit,
				Protection: organizational_units.OUProtectionResult{ProtectedFromAccidentalDeletion: unit.ProtectedFromAccidentalDeletion},
			}, nil
		},
		Update: func(_ context.Context, req organizational_units.UpdateOrganizationalUnitRequest) (organizational_units.OrganizationalUnitMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(req.Identity, req.Options)
			d.calls["update"]++
			if req.Properties.Description != nil {
				d.Description = *req.Properties.Description
				d.descriptionPresent = true
			}
			for _, change := range req.Changes {
				if change.Attribute == "description" && change.Kind == shared.AttributeClear {
					d.descriptionPresent = false
				}
			}
			if req.ProtectedFromAccidentalDeletion != nil {
				d.protect = *req.ProtectedFromAccidentalDeletion
			}
			unit := d.snapshot()
			mutation := d.mutation()
			var err error
			if len(req.Changes) > 0 {
				mutation.Verified = false
				err = shared.ErrObjectUnverifiable
			}
			if req.ProtectedFromAccidentalDeletion == nil {
				unit.ProtectedFromAccidentalDeletion = nil
			}
			return organizational_units.OrganizationalUnitMutationResult{
				Object: mutation, Unit: unit,
				Protection: organizational_units.OUProtectionResult{ProtectedFromAccidentalDeletion: unit.ProtectedFromAccidentalDeletion},
			}, err
		},
		Rename: func(_ context.Context, req shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(req.Identity, req.Options)
			d.calls["rename"]++
			d.Name = req.Name
			return d.mutation(), nil
		},
		Move: func(_ context.Context, req shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(req.Identity, req.Options)
			if req.Destination != nil || req.TargetParent.Form() != shared.IdentityFormDN {
				t.Error("move enabled cross-domain behavior or guessed parent identity")
			}
			d.calls["move"]++
			d.ParentDN = req.TargetParent.String()
			return shared.ObjectMoveResult{Mutation: d.mutation()}, nil
		},
		Delete: func(_ context.Context, req organizational_units.DeleteOrganizationalUnitRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			checkTarget(req.Identity, req.Options)
			if req.Recursive {
				t.Error("delete became recursive")
			}
			d.calls["delete"]++
			result := d.mutation()
			deleted := d.snapshot().Raw
			deleted.Attributes["isdeleted"] = [][]byte{[]byte("TRUE")}
			result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{deleted}}
			d.exists = false
			return result, nil
		},
	}
}
