// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

type labSecretPlan struct{ secrets []string }

// CheckPlan checks the entire Terraform JSON plan for every bind and fixture password, including ephemeral reader input.
func (check labSecretPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = errLabOwnership
		return
	}
	for _, secret := range check.secrets {
		if secret != "" && strings.Contains(string(encoded), secret) {
			resp.Error = errLabOwnership
			return
		}
	}
}

// assertAccount independently observes configured settings on an owned account rather than trusting Terraform state alone.
func (l *accountSettingsLab) assertAccount(subject labSubject, enabled, never bool, homeRequired *bool) error {
	if _, err := l.ownedObject(context.Background(), subject); err != nil {
		return errLabOwnership
	}
	control, err := l.observer.Accounts.AccountControl(context.Background(), l.identity(subject), shared.ResolveIdentityOptions{})
	if err != nil || control.Account == nil || control.Account.Enabled == nil || *control.Account.Enabled != enabled {
		return errLabOwnership
	}
	expiry, err := l.observer.Accounts.AccountExpiration(context.Background(), l.identity(subject), shared.ResolveIdentityOptions{})
	if err != nil || expiry.Expiration == nil || expiry.Expiration.NeverExpires == nil || *expiry.Expiration.NeverExpires != never {
		return errLabOwnership
	}
	if homeRequired != nil && (control.Account.HomeDirectoryRequired == nil || *control.Account.HomeDirectoryRequired != *homeRequired) {
		return errLabOwnership
	}
	return nil
}

// TestAccAccountSettingsLifecycle adopts real user/computer fixtures, repairs only owned drift, imports, clears expiration and relinquishes both intact.
func TestAccAccountSettingsLifecycle(t *testing.T) {
	lab := newAccountSettingsLab(t)
	user := lab.createSubject(t, "user")
	computer := lab.createSubject(t, "computer")
	alternate := lab.createNamedSubject(t, "user", "alternate")
	configuration := func(userSettings, computerSettings string) string {
		return lab.configuration(
			t,
			"resource.tf",
			"__USER_GUID__",
			quotedGUID(user),
			"__COMPUTER_GUID__",
			quotedGUID(computer),
			"# __USER_SETTINGS__",
			userSettings,
			"# __COMPUTER_SETTINGS__",
			computerSettings,
		)
	}
	initial := configuration(`enabled = true
  account_not_delegated = true
  password_never_expires = false
  change_password_at_logon = true
  expires_at = "2031-04-05T11:34:56Z"`, `enabled = true
  account_not_delegated = true
  expires_at = "2031-04-05T11:34:56Z"`)
	updated := configuration(`enabled = true
  account_not_delegated = true
  password_never_expires = false
  change_password_at_logon = false
  never_expires = true`, `enabled = false
  account_not_delegated = true
  never_expires = true`)
	replaced := strings.Replace(updated, quotedGUID(user), quotedGUID(alternate), 1)
	omitted := strings.Replace(configuration("", ""), quotedGUID(user), quotedGUID(alternate), 1)
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	initialProof := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		if err := lab.assertAccount(user, true, false, nil); err != nil {
			return err
		}
		return lab.assertAccount(computer, true, false, nil)
	}
	finalProof := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		home := true
		if err := lab.assertAccount(user, true, true, &home); err != nil {
			return err
		}
		if err := lab.assertAccount(computer, false, true, nil); err != nil {
			return err
		}
		return nil
	}
	replacementProof := func(state *terraform.State) error {
		if err := finalProof(state); err != nil {
			return err
		}
		return lab.assertAccount(alternate, true, true, nil)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(initialProof,
			resource.TestCheckResourceAttr(ResourceName+".user", "id", user.GUID),
			resource.TestCheckResourceAttr(ResourceName+".computer", "id", computer.GUID),
			resource.TestCheckResourceAttr(ResourceName+".user", "change_password_at_logon", "true"),
			resource.TestCheckResourceAttrPair("data."+ResourceName+".by_dn", "distinguished_name", ResourceName+".user", "distinguished_name"),
		)},
		{Config: initial, PlanOnly: true},
		{ResourceName: ResourceName + ".user", ImportState: true, ImportStateVerify: true},
		{ResourceName: ResourceName + ".computer", ImportState: true, ImportStateVerify: true},
		{
			ResourceName:    ResourceName + ".user",
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithResourceIdentity,
			GenerateConfig:  true,
			ImportPlanChecks: resource.ImportPlanChecks{
				PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}, plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop)},
			},
		},
		{Config: updated, ConfigPlanChecks: checks, PreConfig: func() {
			enabled, home := false, true
			lab.mutateFlags(t, user, accounts.AccountControlPatch{Enabled: &enabled, HomeDirectoryRequired: &home})
			lab.renameSubject(t, user)
		}, Check: resource.ComposeAggregateTestCheckFunc(finalProof,
			resource.TestCheckResourceAttr(ResourceName+".user", "id", user.GUID),
			resource.TestCheckResourceAttr(ResourceName+".user", "home_directory_required", "true"),
			resource.TestCheckResourceAttr(ResourceName+".user", "change_password_at_logon", "false"),
			resource.TestCheckResourceAttr(ResourceName+".user", "never_expires", "true"),
			resource.TestCheckNoResourceAttr(ResourceName+".user", "expires_at"),
		)},
		{Config: updated, PlanOnly: true},
		{Config: replaced, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			labSecretPlan{secrets: lab.secrets},
			plancheck.ExpectResourceAction(ResourceName+".user", plancheck.ResourceActionDestroyBeforeCreate),
			plancheck.ExpectResourceAction(ResourceName+".computer", plancheck.ResourceActionNoop),
		}}, Check: resource.ComposeAggregateTestCheckFunc(replacementProof, resource.TestCheckResourceAttr(ResourceName+".user", "id", alternate.GUID))},
		{Config: replaced, PlanOnly: true},
		{Config: omitted, PlanOnly: true},
	}})
	if err := finalProof(nil); err != nil {
		t.Fatal("relinquishment changed or removed independently owned fixture accounts")
	}
	if err := lab.assertAccount(alternate, true, true, nil); err != nil {
		t.Fatal("replacement target was reset or removed during relinquishment")
	}
}

// TestAccAccountSettingsDenied qualifies read-only adoption and refused writes with an ordinary actor while the account remains unchanged.
func TestAccAccountSettingsDenied(t *testing.T) {
	lab := newAccountSettingsLab(t)
	if os.Getenv("TF_VAR_reader_upn") == "" || os.Getenv("TF_VAR_reader_password") == "" {
		t.Fatal("independent ordinary reader credentials are required")
	}
	user := lab.createSubject(t, "user")
	configuration := func(enabled bool) string {
		return lab.configuration(t, "denied.tf", "__USER_GUID__", quotedGUID(user), "__ENABLED__", strconv.FormatBool(enabled))
	}
	baseline := configuration(false)
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	unchanged := func(state *terraform.State) error {
		if state != nil {
			if err := lab.capture(state); err != nil {
				return err
			}
		}
		return lab.assertAccount(user, false, true, nil)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: baseline, ConfigPlanChecks: checks, Check: unchanged},
		{Config: configuration(true), ConfigPlanChecks: checks, ExpectError: regexp.MustCompile("Account settings mutation rejected")},
		{Config: baseline, Check: unchanged},
		{Config: baseline, PlanOnly: true},
	}})
	if err := unchanged(nil); err != nil {
		t.Fatal("refused settings write or relinquishment changed the account")
	}
}
