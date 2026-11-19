// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
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

func (l *computerLab) computerConfig(t *testing.T, name, parent, provider, description, password string, enabled bool, expiration string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(l.prefix))
	sam := fmt.Sprintf("tfc%x", digest[:6])
	return l.configuration(
		t,
		"resource.tf",
		"__NAME__",
		strconv.Quote(l.prefix+"-"+name),
		"__SAM__",
		strconv.Quote(sam+"$"),
		"__UPN__",
		strconv.Quote(sam+"$@ad.sdk.test"),
		"__DNS__",
		strconv.Quote(sam+".ad.sdk.test"),
		"__PARENT__",
		strconv.Quote(parent),
		"__PROVIDER__",
		provider,
		"__DESCRIPTION_BLOCK__",
		description,
		"__PASSWORD_BLOCK__",
		password,
		"__ENABLED__",
		strconv.FormatBool(enabled),
		"__EXPIRATION__",
		expiration,
		"__FILTER__",
		strconv.Quote("(name="+l.prefix+"*)"),
	)
}

// TestAccComputerLifecycle qualifies real Terraform staged creation, rotation, rename/move, import and discovery with secret-free artifacts.
func TestAccComputerLifecycle(t *testing.T) {
	lab := newComputerLab(t)
	initialSecret, rotatedSecret := os.Getenv("TF_VAR_computer_password"), os.Getenv("AD_ACCEPTANCE_ROTATED_COMPUTER_PASSWORD")
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
	initial := lab.computerConfig(
		t,
		"initial",
		lab.parentDN,
		"activedirectory",
		`description = "first description"
 kerberos_encryption_types = ["aes128", "aes256"]
 compound_identity_supported = false`,
		"password_wo = var.computer_password\n password_wo_version = 1",
		true,
		"0",
	)
	updated := lab.computerConfig(
		t,
		"renamed",
		moveDN,
		"activedirectory",
		`description = "second description"
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 change_password_at_logon = true`,
		"password_wo = var.computer_password\n password_wo_version = 2",
		true,
		"9223372036854775807",
	)
	cleared := lab.computerConfig(
		t,
		"renamed",
		moveDN,
		"activedirectory",
		`clear_properties = ["description"]
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 change_password_at_logon = true`,
		"password_wo = var.computer_password\n password_wo_version = 2",
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
			PreConfig:        func() { createDestination(); t.Setenv("TF_VAR_computer_password", rotatedSecret) },
			ConfigPlanChecks: planChecks,
			Check: resource.ComposeAggregateTestCheckFunc(
				check,
				resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "2"),
				resource.TestCheckResourceAttr("data.activedirectory_computers.scoped", "computers.#", "1"),
				resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "true"),
				resource.TestCheckResourceAttr(ResourceName+".test", "kerberos_encryption_types.#", "1"),
				resource.TestCheckResourceAttr(ResourceName+".test", "compound_identity_supported", "true"),
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

// TestAccComputerPasswordObservation qualifies pwdLastSet observation and explicit post-reset intent on a fresh disabled, unjoined account.
func TestAccComputerPasswordObservation(t *testing.T) {
	lab := newComputerLab(t)
	initialSecret, rotatedSecret := os.Getenv("TF_VAR_computer_password"), os.Getenv("AD_ACCEPTANCE_ROTATED_COMPUTER_PASSWORD")
	if initialSecret == "" || rotatedSecret == "" || initialSecret == rotatedSecret {
		t.Fatal("two distinct strong fixture-only passwords must be supplied through protected process environment")
	}
	initial := lab.computerConfig(t, "disabled", lab.parentDN, "activedirectory", `description = "disabled unjoined fixture"`, "", false, "0")
	rotated := lab.computerConfig(t, "disabled", lab.parentDN, "activedirectory", `description = "disabled unjoined fixture"`,
		"password_wo = var.computer_password\n password_wo_version = 1", false, "0")
	explicit := lab.computerConfig(t, "disabled", lab.parentDN, "activedirectory", `description = "disabled unjoined fixture"
 change_password_at_logon = true`, "password_wo = var.computer_password\n password_wo_version = 2", false, "0")
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: []string{initialSecret, rotatedSecret}}}}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "false"),
			resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "true"))},
		{Config: initial, PlanOnly: true},
		{Config: rotated, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "false"),
			resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "1"))},
		{Config: rotated, PlanOnly: true},
		{
			Config: explicit, PreConfig: func() { t.Setenv("TF_VAR_computer_password", rotatedSecret) }, ConfigPlanChecks: checks,
			Check: resource.ComposeAggregateTestCheckFunc(lab.capture, resource.TestCheckResourceAttr(ResourceName+".test", "change_password_at_logon", "true"),
				resource.TestCheckResourceAttr(ResourceName+".test", "password_wo_version", "2"), resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "false")),
		},
		{Config: explicit, PlanOnly: true},
	}})
}
