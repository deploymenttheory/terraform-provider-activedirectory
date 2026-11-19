// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// objectBindSecretPlan checks real plan artifacts without printing the credential or plan on failure.
type objectBindSecretPlan struct{}

func (objectBindSecretPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = errLabOwnership
		return
	}
	if secret := os.Getenv("AD_PASSWORD"); secret != "" && strings.Contains(string(encoded), secret) {
		resp.Error = errLabOwnership
	}
}

// objectConfig uses JSON-compatible HCL values to preserve opaque raw bytes and literal names in embedded fixtures.
func (l *objectLab) objectConfig(t *testing.T, suffix, class, parent string, attributes, add, remove map[string][]string, clear []string) string {
	t.Helper()
	encode := func(value any) string {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal("fixture HCL value could not be encoded")
		}
		return string(encoded)
	}
	name := l.prefix + "-" + suffix
	return l.configuration(
		t,
		"resource.tf",
		"__NAME__",
		strconv.Quote(name),
		"__CLASS__",
		strconv.Quote(class),
		"__PARENT__",
		strconv.Quote(parent),
		"__ALL_FILTER__",
		strconv.Quote("(name="+l.prefix+"*)"),
		"__ATTRIBUTES__",
		encode(attributes),
		"__ADD_VALUES__",
		encode(add),
		"__REMOVE_VALUES__",
		encode(remove),
		"__CLEAR_ATTRIBUTES__",
		encode(clear),
		"__FILTER__",
		strconv.Quote("(&(objectClass="+class+")(name="+name+"))"),
	)
}

// encodedText expresses raw fixture strings explicitly rather than asking the provider to infer an LDAP syntax.
func encodedText(text string) string { return base64.StdEncoding.EncodeToString([]byte(text)) }

// TestAccObjectLifecycle qualifies binary/raw ownership, stable import, normalized DNs, discovery and drift against real AD DS.
func TestAccObjectLifecycle(t *testing.T) {
	lab := newObjectLab(t)
	moveDN := "OU=" + lab.prefix + "-move," + lab.parentDN
	managerDN := "CN=" + lab.prefix + "-manager," + lab.parentDN
	var managerGUID string
	initialAttributes := map[string][]string{
		"description": {encodedText(lab.marker)}, "displayName": {encodedText("first display")}, "thumbnailPhoto": {"/wCA"},
	}
	updatedAttributes := map[string][]string{
		"description": {encodedText(lab.marker)}, "displayName": {encodedText("second display")}, "thumbnailPhoto": {"AP+A"},
		"manager": {encodedText(strings.ToLower(managerDN))},
	}
	clearedAttributes := map[string][]string{
		"description": {encodedText(lab.marker)}, "thumbnailPhoto": {"AP+A"}, "manager": {encodedText(strings.ToLower(managerDN))},
	}
	initial := lab.objectConfig(t, "initial", "contact", lab.parentDN, initialAttributes,
		map[string][]string{"otherTelephone": {encodedText("old"), encodedText("keep")}}, nil, nil)
	updated := lab.objectConfig(t, "renamed", "contact", moveDN, updatedAttributes,
		map[string][]string{"otherTelephone": {encodedText("new")}}, map[string][]string{"otherTelephone": {encodedText("old")}}, nil)
	cleared := lab.objectConfig(t, "renamed", "contact", moveDN, clearedAttributes,
		map[string][]string{"otherTelephone": {encodedText("new")}}, map[string][]string{"otherTelephone": {encodedText("old")}}, []string{"displayName"})
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__",
		strconv.Quote("(&(objectClass=contact)(name="+lab.prefix+"-renamed))"))
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{objectBindSecretPlan{}}}
	check := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		secret := os.Getenv("AD_PASSWORD")
		for _, module := range state.Modules {
			for _, entry := range module.Resources {
				if entry.Primary != nil {
					for _, value := range entry.Primary.Attributes {
						if secret != "" && strings.Contains(value, secret) {
							return errLabOwnership
						}
					}
				}
			}
		}
		return nil
	}
	prepareUpdate := func() {
		lab.createAuxiliary(t, "organizationalUnit", "move", lab.parentGUID)
		managerGUID = lab.createAuxiliary(t, "contact", "manager", lab.parentGUID)
		lab.mutateOwnedValues(t, []shared.AttributeChange{
			{Attribute: "otherTelephone", Kind: shared.AttributeAddValues, Values: []shared.RawValue{shared.NewRawValue([]byte("unowned"))}},
		})
	}
	verifyManager := func(state *terraform.State) error {
		if err := check(state); err != nil {
			return err
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		object, err := lab.ownedObject(t.Context(), lab.childGUID)
		if err != nil {
			return errLabOwnership
		}
		value, err := object.Values("manager").Single()
		if err != nil {
			return errLabOwnership
		}
		target, err := shared.IdentityByDN(string(value.Bytes()))
		if err != nil {
			return errLabOwnership
		}
		manager, err := lab.observer.Directory.ResolveIdentity(t.Context(), target, shared.ResolveIdentityOptions{})
		if err != nil || !manager.Complete || manager.Object == nil {
			return errLabOwnership
		}
		guid, err := manager.Object.Values("objectGUID").Single()
		if err != nil {
			return errLabOwnership
		}
		parsed, err := guid.GUID()
		if err != nil || parsed.String() != managerGUID {
			return errLabOwnership
		}
		return nil
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(check,
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.thumbnailphoto.*", "/wCA"))},
		{Config: initial, PlanOnly: true},
		{Config: updated, PreConfig: prepareUpdate, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(verifyManager,
			resource.TestCheckResourceAttr("data.activedirectory_objects.scoped", "objects.#", "1"),
			resource.TestCheckResourceAttr("data.activedirectory_objects.paged", "objects.#", "3"),
			resource.TestCheckResourceAttr("data.activedirectory_objects.paged", "complete", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_objects.limited", "objects.#", "1"),
			resource.TestCheckResourceAttr("data.activedirectory_objects.limited", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_objects.limited", "complete", "false"),
			resource.TestCheckResourceAttrPair(ResourceName+".test", "id", "data.activedirectory_object.by_dn", "id"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", encodedText("keep")),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", encodedText("unowned")),
			resource.TestCheckTypeSetElemAttr("data.activedirectory_object.lookup", "observed_attributes.thumbnailphoto.*", "AP+A"))},
		{Config: updated, PlanOnly: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true,
			ImportStateIdFunc: func(_ *terraform.State) (string, error) {
				encoded, err := json.Marshal(importSelection{
					ID: lab.childGUID, Attributes: []string{"description", "displayName", "thumbnailPhoto", "manager"},
					Properties: []string{"description", "displayName", "thumbnailPhoto", "manager", "otherTelephone", "telephoneNumber"},
				})
				return string(encoded), err
			},
			ImportStateVerifyIgnore: []string{"add_values", "remove_values", "attributes.manager"},
		},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
				objectBindSecretPlan{}, plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
			}},
		},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)}},
		{Config: cleared, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(check,
			resource.TestCheckResourceAttr(ResourceName+".test", "attribute_status.displayname", "absent"),
			resource.TestCheckNoResourceAttr(ResourceName+".test", "observed_attributes.displayname"))},
		{Config: cleared, PlanOnly: true},
		{Config: cleared, ConfigPlanChecks: checks, PreConfig: func() {
			lab.mutateOwnedValues(t, []shared.AttributeChange{
				{Attribute: "otherTelephone", Kind: shared.AttributeRemoveValues, Values: []shared.RawValue{shared.NewRawValue([]byte("new"))}},
				{
					Attribute: "otherTelephone",
					Kind:      shared.AttributeAddValues,
					Values:    []shared.RawValue{shared.NewRawValue([]byte("old")), shared.NewRawValue([]byte("drift-unowned"))},
				},
			})
		}, Check: resource.ComposeAggregateTestCheckFunc(verifyManager,
			resource.TestCheckResourceAttr(ResourceName+".test", "needs_update", "false"),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", encodedText("new")),
			resource.TestCheckTypeSetElemAttr(ResourceName+".test", "observed_attributes.othertelephone.*", encodedText("drift-unowned")))},
		{Config: cleared, PlanOnly: true},
	}})
}

// TestAccObjectNonemptyLeaf qualifies schema-derived OU naming and refuses recursive removal before explicit child retirement.
func TestAccObjectNonemptyLeaf(t *testing.T) {
	lab := newObjectLab(t)
	configuration := lab.objectConfig(t, "container", "organizationalUnit", lab.parentDN,
		map[string][]string{"description": {encodedText(lab.marker)}}, nil, nil, nil)
	childGUID := ""
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: configuration, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "distinguished_name", "OU="+lab.prefix+"-container,"+lab.parentDN))},
		{Config: configuration, Destroy: true, PreConfig: func() {
			childGUID = lab.createAuxiliary(t, "contact", "child", lab.childGUID)
		}, ExpectError: regexp.MustCompile("Object deletion not confirmed")},
		{Config: configuration, PreConfig: func() {
			if err := lab.removeAuxiliary(t.Context(), childGUID); err != nil {
				t.Fatal("independent owned child retirement did not verify")
			}
		}, Check: lab.capture},
		{Config: configuration, PlanOnly: true},
	}})
}
