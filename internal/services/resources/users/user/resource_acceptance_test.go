// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

type labSecretPlan struct{ secrets []string }

func (c labSecretPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	for _, secret := range c.secrets {
		if secret != "" && strings.Contains(string(encoded), secret) {
			resp.Error = errSecretInArtifact
			return
		}
	}
}

func (l *userLab) userConfig(t *testing.T, name, parent, provider, description, password string, enabled bool, expiration string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(l.prefix))
	sam := fmt.Sprintf("tfu%x", digest[:6])
	return l.configuration(t, "resource.tf", "__NAME__", strconv.Quote(l.prefix+"-"+name), "__SAM__", strconv.Quote(sam), "__UPN__", strconv.Quote(sam+"@ad.sdk.test"),
		"__PARENT__", strconv.Quote(parent), "__PROVIDER__", provider, "__DESCRIPTION_BLOCK__", description, "__PASSWORD_BLOCK__", password,
		"__ENABLED__", strconv.FormatBool(enabled), "__EXPIRATION__", expiration, "__FILTER__", strconv.Quote("(name="+l.prefix+"*)"))
}

// TestAccUserLifecycle qualifies real Terraform staged creation, rotation, rename/move, import and discovery with secret-free artifacts.
func TestAccUserLifecycle(t *testing.T) {
	lab := newUserLab(t)
	initialSecret, rotatedSecret := os.Getenv("TF_VAR_user_password"), os.Getenv("AD_ACCEPTANCE_ROTATED_USER_PASSWORD")
	if initialSecret == "" || rotatedSecret == "" || initialSecret == rotatedSecret {
		t.Fatal("two distinct strong fixture-only passwords must be supplied through protected process environment")
	}
	moveDN := "OU=" + lab.prefix + "-move," + lab.parentDN
	var moveGUID string
	createDestination := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil {
			t.Fatal("independent observer not initialized")
		}
		parent, _ := shared.IdentityByGUID(lab.parentGUID)
		protect := false
		description := "tfad-move:" + lab.target.RunID
		result, err := lab.observer.OrganizationalUnits.CreateOrganizationalUnit(t.Context(), organizational_units.CreateOrganizationalUnitRequest{
			Parent:                          parent,
			Name:                            lab.prefix + "-move",
			ProtectedFromAccidentalDeletion: &protect,
			Properties:                      organizational_units.OUProperties{DisplayName: &lab.marker, Description: &description},
		})
		if result.Object.TargetGUID != nil {
			moveGUID = result.Object.TargetGUID.String()
		}
		if err != nil || !result.Object.Verified || moveGUID == "" {
			t.Fatal("owned move destination did not verify; preserve partial outcome")
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
			t.Error("move destination ownership did not verify")
			return
		}
		deleted, err := lab.observer.OrganizationalUnits.DeleteOrganizationalUnit(
			context.Background(),
			organizational_units.DeleteOrganizationalUnitRequest{Identity: target, Recursive: false},
		)
		if !lifecycle.ConfirmedDeletion(deleted, err) {
			t.Error("owned move destination deletion did not verify")
		}
	})
	initial := lab.userConfig(
		t,
		"initial",
		lab.parentDN,
		"activedirectory",
		`description = "first description"`,
		"password_wo = var.user_password\n password_wo_version = 1",
		true,
		"0",
	)
	updated := lab.userConfig(
		t,
		"renamed",
		moveDN,
		"activedirectory",
		`description = "second description"`,
		"password_wo = var.user_password\n password_wo_version = 2",
		true,
		"9223372036854775807",
	)
	cleared := lab.userConfig(
		t,
		"renamed",
		moveDN,
		"activedirectory",
		`clear_properties = ["description"]`,
		"password_wo = var.user_password\n password_wo_version = 2",
		true,
		"9223372036854775807",
	)
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"))
	planChecks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: []string{initialSecret, rotatedSecret}}}}
	check := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		for _, module := range state.Modules {
			for _, object := range module.Resources {
				if object.Primary != nil {
					for _, value := range object.Primary.Attributes {
						if strings.Contains(value, initialSecret) || strings.Contains(value, rotatedSecret) {
							return errSecretInArtifact
						}
					}
				}
			}
		}
		return nil
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{
			Config:           initial,
			ConfigPlanChecks: planChecks,
			Check:            resource.ComposeAggregateTestCheckFunc(check, resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "true")),
		},
		{Config: initial, PlanOnly: true},
		{
			Config:           updated,
			PreConfig:        func() { createDestination(); t.Setenv("TF_VAR_user_password", rotatedSecret) },
			ConfigPlanChecks: planChecks,
			Check: resource.ComposeAggregateTestCheckFunc(
				check,
				resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "2"),
				resource.TestCheckResourceAttr("data.activedirectory_users.scoped", "users.#", "1"),
			),
		},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{
			ResourceName:    ResourceName + ".test",
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithResourceIdentity,
			GenerateConfig:  true,
			ImportPlanChecks: resource.ImportPlanChecks{
				PreApply: []plancheck.PlanCheck{
					labSecretPlan{secrets: []string{initialSecret, rotatedSecret}},
					plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
				},
			},
		},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)}},
		{
			Config:           cleared,
			ConfigPlanChecks: planChecks,
			Check:            resource.ComposeAggregateTestCheckFunc(check, resource.TestCheckNoResourceAttr(ResourceName+".test", "description")),
		},
		{Config: cleared, PlanOnly: true},
	}})
}

// TestAccUserPasswordFailure preserves the accepted disabled creation and refuses automatic replacement after password rejection.
func TestAccUserPasswordFailure(t *testing.T) {
	lab := newUserLab(t)
	t.Setenv("TF_VAR_user_password", "bad")
	configuration := lab.userConfig(
		t,
		"rejected",
		lab.parentDN,
		"activedirectory",
		`description = "rejected password fixture"`,
		"password_wo = var.user_password\n password_wo_version = 1",
		true,
		"0",
	)
	retire := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil {
			t.Fatal("independent observer unavailable")
		}
		result, err := lab.observer.Users.Users(
			t.Context(),
			shared.SearchOptions{Base: lab.parentDN, Scope: shared.ScopeSubtree, Filter: "(name=" + lab.prefix + "*)", PageSize: 10, ResultLimit: 10},
		)
		if err != nil || !result.Observation.Complete || len(result.Users) != 1 {
			t.Fatal("accepted creation could not be reconciled uniquely")
		}
		user := result.Users[0]
		if user.GUID == nil || user.Enabled == nil || *user.Enabled || user.DisplayName == nil || *user.DisplayName != lab.marker {
			t.Fatal("accepted failed creation lost ownership or disabled status")
		}
		lab.guids[user.GUID.String()] = true
		target, _ := shared.IdentityByGUID(user.GUID.String())
		deleted, err := lab.observer.Users.DeleteUser(t.Context(), users.DeleteUserRequest{Identity: target})
		if !lifecycle.ConfirmedDeletion(deleted.Mutation, err) {
			t.Fatal("manual retirement of accepted disabled fixture did not verify")
		}
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: configuration, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
		{Config: configuration, ExpectError: regexp.MustCompile("User mutation still unresolved")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{Config: configuration, Destroy: true, PreConfig: retire},
	}})
}
