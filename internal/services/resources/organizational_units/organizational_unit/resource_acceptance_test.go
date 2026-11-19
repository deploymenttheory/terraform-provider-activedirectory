// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnit

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

// TestAccOrganizationalUnitLifecycle qualifies real Terraform CRUD, discovery and GUID import in the selected owned lab.
func TestAccOrganizationalUnitLifecycle(t *testing.T) {
	lab := newOULab(t)
	configuration := func(name, parent, description string, protect bool) string {
		return lab.configuration(t, "lifecycle.tf", "__NAME__", name, "__TARGET_PARENT__", ResourceName+"."+parent+".distinguished_name",
			"__DESCRIPTION_BLOCK__", description, "__PROTECTED__", strconv.FormatBool(protect))
	}
	initial := configuration("child", "parent_a", `description = "first description"`, true)
	updated := configuration("renamed", "parent_b", `description = "second description"`, false)
	cleared := configuration("renamed", "parent_b", `clear_properties = ["description"]`, false)
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy,
		Steps: []resource.TestStep{
			{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
				resource.TestCheckResourceAttr(ResourceName+".test", "protected_from_accidental_deletion", "true"),
				resource.TestCheckResourceAttr("data.activedirectory_organizational_units.scoped", "organizational_units.#", "3"))},
			{Config: initial, PlanOnly: true},
			{Config: updated, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
				resource.TestCheckResourceAttr(ResourceName+".test", "name", lab.prefix+"-renamed"),
				resource.TestCheckResourceAttr(ResourceName+".test", "protected_from_accidental_deletion", "false"),
				resource.TestCheckResourceAttr("data.activedirectory_organizational_unit.lookup", "description", "second description"))},
			{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
			{
				ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
				ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
					plancheck.ExpectKnownValue(ResourceName+".generated", tfjsonpath.New("name"), knownvalue.StringExact(lab.prefix+"-renamed")),
				}},
			},
			{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 3)}},
			{Config: cleared, Check: resource.ComposeAggregateTestCheckFunc(lab.capture, resource.TestCheckNoResourceAttr(ResourceName+".test", "description"))},
			{Config: cleared, PlanOnly: true},
		},
	})
}

// TestAccOrganizationalUnitProtectedDelete preserves a rejected protected OU until explicit unprotection is applied.
func TestAccOrganizationalUnitProtectedDelete(t *testing.T) {
	lab := newOULab(t)
	configuration := func(protect bool) string {
		return lab.configuration(t, "single.tf", "__NAME__", strconv.Quote(lab.prefix+"-protected"), "__PROTECTED__", strconv.FormatBool(protect))
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: configuration(true), Check: lab.capture},
		{Config: configuration(true), Destroy: true, ExpectError: regexp.MustCompile("OU deletion not confirmed")},
		{
			Config: configuration(false),
			Check:  resource.ComposeAggregateTestCheckFunc(lab.capture, resource.TestCheckResourceAttr(ResourceName+".test", "protected_from_accidental_deletion", "false")),
		},
	}})
}

// TestAccOrganizationalUnitNonemptyDelete verifies Terraform never turns a leaf deletion into subtree removal.
func TestAccOrganizationalUnitNonemptyDelete(t *testing.T) {
	lab := newOULab(t)
	configuration := lab.configuration(t, "single.tf", "__NAME__", strconv.Quote(lab.prefix+"-nonempty"), "__PROTECTED__", "false")
	var contactGUID string
	createChild := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil || lab.childGUID == "" {
			t.Fatal("owned parent identity was not recorded")
		}
		parent, err := shared.IdentityByGUID(lab.childGUID)
		if err != nil {
			t.Fatal("invalid owned parent GUID")
		}
		result, err := lab.observer.Directory.CreateObject(t.Context(), shared.CreateObjectRequest{
			Parent: parent,
			Class:  "contact",
			Name:   lab.prefix + "-contact",
			Attributes: map[string][]shared.RawValue{
				"displayName": {shared.NewRawValue([]byte(lab.marker))},
				"description": {shared.NewRawValue([]byte("sdk-portable-owned:" + lab.target.RunID))},
			},
		})
		if result.TargetGUID != nil {
			contactGUID = result.TargetGUID.String()
			lab.guids[contactGUID] = true
		}
		if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified || contactGUID == "" {
			t.Fatal("owned nonempty fixture creation did not verify; preserve partial outcome")
		}
	}
	removeChild := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		identity, err := shared.IdentityByGUID(contactGUID)
		if err != nil {
			t.Fatal("owned contact GUID missing")
		}
		observed, err := lab.observer.Directory.ResolveIdentity(t.Context(), identity, shared.ResolveIdentityOptions{Properties: []string{"displayName", "description"}})
		if err != nil || !observed.Complete || observed.Object == nil {
			t.Fatal("owned contact read did not verify")
		}
		marker, markerErr := observed.Object.Values("displayName").Single()
		if markerErr != nil || string(marker.Bytes()) != lab.marker {
			t.Fatal("refusing to remove a contact with changed ownership")
		}
		result, err := lab.observer.Directory.DeleteObject(context.Background(), shared.DeleteObjectRequest{Identity: identity, Recursive: false})
		if !lifecycle.ConfirmedDeletion(result, err) {
			t.Fatal("owned contact deletion did not independently verify")
		}
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: configuration, Check: lab.capture},
		{Config: configuration, PreConfig: createChild, Destroy: true, ExpectError: regexp.MustCompile("OU deletion not confirmed")},
		{Config: configuration, PreConfig: removeChild, Check: lab.capture},
	}})
}

// TestAccOrganizationalUnitDeniedAndHidden preserves managed state when an ordinary reader cannot observe or change a property.
func TestAccOrganizationalUnitDeniedAndHidden(t *testing.T) {
	lab := newOULab(t)
	guid := os.Getenv("AD_ACCEPTANCE_HIDDEN_GUID")
	name := os.Getenv("AD_ACCEPTANCE_HIDDEN_NAME")
	if _, err := shared.IdentityByGUID(guid); err != nil || name == "" || os.Getenv("TF_VAR_reader_upn") == "" || os.Getenv("TF_VAR_reader_password") == "" {
		t.Fatal("independently prepared owned permission fixture and ordinary reader credentials are required")
	}
	baseline := "tfad-hidden:" + lab.target.RunID
	configuration := func(provider, description, imported, lookup string) string {
		return lab.configuration(t, "permissions.tf", "__PROVIDER__", provider, "__NAME__", strconv.Quote(name),
			"__DESCRIPTION__", strconv.Quote(description), "__IMPORT_BLOCK__", imported, "__LOOKUP_BLOCK__", lookup)
	}
	admin := configuration("activedirectory", baseline, "", "")
	imported := configuration("activedirectory", baseline, "import {\n  to = "+ResourceName+".test\n  id = "+strconv.Quote(guid)+"\n}\n", "")
	lookup := "data \"activedirectory_organizational_unit\" \"hidden\" {\n  provider = activedirectory.reader\n  object_guid = " + strconv.Quote(guid) + "\n}\n"
	reader := configuration("activedirectory.reader", baseline, "", lookup)
	verifyBaseline := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		identity, _ := shared.IdentityByGUID(guid)
		result, err := lab.observer.OrganizationalUnits.OrganizationalUnit(t.Context(), identity, shared.ResolveIdentityOptions{})
		if err != nil || !result.Observation.Complete || result.Unit == nil || result.Unit.Description == nil || *result.Unit.Description != baseline {
			return errLabOwnership
		}
		return nil
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: imported, Check: verifyBaseline},
		{Config: configuration("activedirectory.reader", "refused change", "", lookup), ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: reader, Check: resource.ComposeAggregateTestCheckFunc(verifyBaseline,
			resource.TestCheckResourceAttr(ResourceName+".test", "id", guid),
			resource.TestCheckResourceAttr(ResourceName+".test", "description", baseline),
			resource.TestCheckNoResourceAttr("data.activedirectory_organizational_unit.hidden", "description"))},
		{Config: reader, PlanOnly: true},
		{Config: admin, Check: verifyBaseline},
	}})
}
