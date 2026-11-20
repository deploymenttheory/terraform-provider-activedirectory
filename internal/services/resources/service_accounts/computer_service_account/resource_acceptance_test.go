// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
)

// TestAccComputerServiceAccountLifecycle qualifies standalone managed-account associations through actual Terraform operations.
func TestAccComputerServiceAccountLifecycle(t *testing.T) {
	testComputerServiceAccountLifecycle(t, sdkaccounts.ServiceAccountStandalone)
}

// TestAccComputerServiceAccountGroupManaged qualifies the same lifecycle using DC-managed accounts and mature KDS prerequisites.
func TestAccComputerServiceAccountGroupManaged(t *testing.T) {
	testComputerServiceAccountLifecycle(t, sdkaccounts.ServiceAccountGroupManaged)
}

// testComputerServiceAccountLifecycle owns isolated fixture endpoints while Terraform owns only the selected directory associations.
func testComputerServiceAccountLifecycle(t *testing.T, kind sdkaccounts.ServiceAccountKind) {
	lab := newComputerServiceAccountLab(t, kind)
	computer := lab.createNamedSubject(t, "computer", "owner")
	solo := lab.createNamedSubject(t, "computer", "solo")
	empty := lab.createNamedSubject(t, "computer", "empty")
	class := "msDS-ManagedServiceAccount"
	if kind == sdkaccounts.ServiceAccountGroupManaged {
		class = "msDS-GroupManagedServiceAccount"
	}
	account := lab.createNamedSubject(t, class, "adopted")
	second := lab.createNamedSubject(t, class, "added")
	unmanaged := lab.createNamedSubject(t, class, "unmanaged")
	alternate := lab.createNamedSubject(t, class, "alternate")
	soloAccount := lab.createNamedSubject(t, class, "soloaccount")
	lab.setLink(t, computer, unmanaged, true)
	lab.setLink(t, computer, account, true)
	initial := lab.configuration(t, "resource.tf",
		"__COMPUTER_GUID__", computer.GUID, "__ACCOUNT_GUID__", account.GUID, "__SECOND_GUID__", second.GUID,
		"__SOLO_COMPUTER_GUID__", solo.GUID, "__SOLO_ACCOUNT_GUID__", soloAccount.GUID, "__EMPTY_COMPUTER_GUID__", empty.GUID,
	)
	replaced := strings.Replace(initial, account.GUID, alternate.GUID, 1)
	query := lab.configuration(t, "list-resource.tfquery.hcl", "__COMPUTER_GUID__", computer.GUID)
	planChecks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: lab.secrets}}}
	proof := func(selected labSubject) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			if err := lab.capture(state); err != nil {
				return err
			}
			for _, subject := range []labSubject{selected, second, unmanaged} {
				if err := lab.assertLink(computer, subject, true); err != nil {
					return err
				}
			}
			return lab.assertLink(solo, soloAccount, true)
		}
	}
	destroyProof := func(state *terraform.State) error {
		if err := lab.checkDestroy(state); err != nil {
			return err
		}
		for _, subject := range []labSubject{account, alternate, second} {
			if err := lab.assertLink(computer, subject, false); err != nil {
				return err
			}
		}
		if err := lab.assertLink(computer, unmanaged, true); err != nil {
			return err
		}
		return lab.assertLink(solo, soloAccount, false)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: destroyProof, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: planChecks, Check: resource.ComposeAggregateTestCheckFunc(proof(account),
			resource.TestCheckResourceAttr(ResourceName+".adopted", "id", computer.GUID+"/"+account.GUID),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.owned", "service_accounts.#", "3"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.owned", "complete", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.bounded", "complete", "false"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.bounded", "limit_reached", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.empty", "complete", "false"),
			resource.TestCheckResourceAttr("data.activedirectory_computer_service_accounts.empty", "service_accounts.#", "0"),
		)},
		{Config: initial, PlanOnly: true},
		{ResourceName: ResourceName + ".adopted", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".solo", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
			}},
		},
		{Query: true, Config: query, QueryResultChecks: []querycheck.QueryResultCheck{
			querycheck.ExpectLength(ResourceName+".owned", 3),
		}},
		{Config: initial, ConfigPlanChecks: planChecks, PreConfig: func() {
			lab.setLink(t, computer, account, false)
			lab.renameSubject(t, computer)
			lab.renameSubject(t, account)
		}, Check: proof(account)},
		{Config: initial, PlanOnly: true},
		{Config: replaced, ConfigPlanChecks: planChecks, Check: resource.ComposeAggregateTestCheckFunc(proof(alternate), func(*terraform.State) error {
			return lab.assertLink(computer, account, false)
		})},
		{Config: replaced, PlanOnly: true},
	}})
}
