// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
)

// TestAccComputerServiceAccountDeniedAndHidden verifies ordinary-actor refusal and distinguishes hidden direct values from an empty computer.
func TestAccComputerServiceAccountDeniedAndHidden(t *testing.T) {
	lab := newComputerServiceAccountLab(t, sdkaccounts.ServiceAccountStandalone)
	computer := lab.createNamedSubject(t, "computer", "denied")
	account := lab.createNamedSubject(t, "msDS-ManagedServiceAccount", "account")
	unmanaged := lab.createNamedSubject(t, "msDS-ManagedServiceAccount", "unmanaged")
	lab.setLink(t, computer, unmanaged, true)
	hidden, hiddenMember, hiddenReadable := lab.hiddenFixture(t)
	configuration := func(attempt string) string {
		return lab.configuration(t, "denied.tf", "__COMPUTER_GUID__", computer.GUID, "__ACCOUNT_GUID__", account.GUID,
			"__UNMANAGED_GUID__", unmanaged.GUID, "# __ATTEMPT_BLOCK__", "count = "+attempt)
	}
	baseline := configuration("0")
	unchanged := func(state *terraform.State) error {
		if err := lab.capture(state); err != nil {
			return err
		}
		if err := lab.assertLink(computer, account, false); err != nil {
			return err
		}
		if err := lab.assertLink(computer, unmanaged, true); err != nil {
			return err
		}
		return lab.assertLink(hidden, hiddenMember, true)
	}
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	hiddenLookup := baseline + lab.configuration(t, "hidden.tf", "__COMPUTER_GUID__", hidden.GUID, "__ACCOUNT_GUID__", hiddenMember.GUID)
	hiddenStep := resource.TestStep{Config: hiddenLookup}
	if hiddenReadable {
		hiddenStep.Check = resource.TestCheckResourceAttr("data.activedirectory_computer_service_account.hidden", "id", hidden.GUID+"/"+hiddenMember.GUID)
	} else {
		hiddenStep.ExpectError = regexp.MustCompile("Direct association unavailable")
	}
	hiddenCollection := baseline + lab.configuration(t, "hidden-collection.tf", "__COMPUTER_GUID__", hidden.GUID)
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: baseline, ConfigPlanChecks: checks, Check: unchanged},
		{Config: configuration("1"), ConfigPlanChecks: checks, ExpectError: regexp.MustCompile("Association mutation rejected")},
		{Config: baseline, Check: unchanged},
		hiddenStep,
		{Config: hiddenCollection, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.hidden", "complete", "false"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.hidden", "service_accounts.#", "0"),
		)},
		{Config: baseline, Check: unchanged},
		{Config: baseline, PlanOnly: true},
	}})
	if err := unchanged(nil); err != nil {
		t.Fatal("refused or hidden association operations changed native directory links")
	}
}
