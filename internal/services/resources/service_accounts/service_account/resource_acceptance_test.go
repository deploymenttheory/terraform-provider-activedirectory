// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
)

type labSecretPlan struct{ secrets []string }

func (c labSecretPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = errLabOwnership
		return
	}
	for _, secret := range c.secrets {
		if secret != "" && strings.Contains(string(encoded), secret) {
			resp.Error = errLabOwnership
			return
		}
	}
}

// TestAccServiceAccountPasswordFailure verifies the cmdlet-compatible disabled partial creation and blocks blind replacement after password rejection.
func TestAccServiceAccountPasswordFailure(t *testing.T) {
	lab := newServiceAccountLab(t)
	lab.secret = "not-complex-enough"
	t.Setenv("TF_VAR_service_account_password", lab.secret)
	configuration := lab.configuration(t, "rejected-password.tf", "__NAME__", strconv.Quote(lab.prefix+"-rejected"), "__SAM__", strconv.Quote(lab.sam("rejected")+"$"))
	observe := func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		result, err := lab.observer.ServiceAccounts.ServiceAccounts(t.Context(), shared.SearchOptions{
			Base: lab.parentDN, Scope: shared.ScopeSubtree, Filter: "(name=" + lab.prefix + "*)", PageSize: 2, ResultLimit: 2,
		})
		if err != nil || !result.Observation.Complete || result.Observation.LimitReached || len(result.Accounts) != 1 {
			t.Fatal("partial creation did not resolve to exactly one owned account")
		}
		account := result.Accounts[0]
		if account.GUID == nil || account.Kind != service_accounts.ServiceAccountStandalone || account.AccountControl.Enabled == nil || *account.AccountControl.Enabled {
			t.Fatal("rejected password creation did not retain a disabled standalone account")
		}
		if _, err := lab.ownedObject(t.Context(), account.GUID.String()); err != nil {
			t.Fatal("partial creation ownership did not verify")
		}
		if lab.childGUID != "" && lab.childGUID != account.GUID.String() {
			t.Fatal("failed creation was replayed or replaced")
		}
		lab.childGUID = account.GUID.String()
		lab.guids[lab.childGUID] = true
	}
	retire := func() {
		observe()
		lab.mu.Lock()
		defer lab.mu.Unlock()
		target, _ := shared.IdentityByGUID(lab.childGUID)
		result, err := lab.observer.ServiceAccounts.DeleteServiceAccount(t.Context(), service_accounts.DeleteServiceAccountRequest{Identity: target})
		if !lifecycle.ConfirmedDeletion(result.Mutation, err) {
			t.Fatal("independent partial-creation fixture retirement did not verify")
		}
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{
			Config:           configuration,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: []string{os.Getenv("AD_PASSWORD"), lab.secret}}}},
			ExpectError:      regexp.MustCompile("Directory mutation did not complete"),
		},
		{Config: configuration, PreConfig: observe, ExpectError: regexp.MustCompile("Service account mutation still unresolved")},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password_wo_version"}},
		{Config: configuration, Destroy: true, PreConfig: retire},
	}})
}

// sam uses a bounded hash rather than truncating a long fixture name into a colliding sAMAccountName.
func (l *serviceAccountLab) sam(suffix string) string {
	digest := sha256.Sum256([]byte(l.prefix + "/" + suffix))
	return fmt.Sprintf("tfs%x", digest[:6])
}

func (l *serviceAccountLab) serviceConfig(t *testing.T, name, parent, attributes string) string {
	t.Helper()
	domain := strings.SplitN(l.target.Host, ".", 2)
	if len(domain) != 2 {
		t.Fatal("explicit lab host domain is unavailable")
	}
	return l.configuration(t, "resource.tf",
		"__NAME__", strconv.Quote(l.prefix+"-"+name), "__SAM__", strconv.Quote(l.sam("subject")+"$"), "__PARENT__", strconv.Quote(parent),
		"__DNS__", strconv.Quote(l.sam("subject")+"."+domain[1]), "__ATTRIBUTES__", attributes, "__FILTER__", strconv.Quote("(name="+l.prefix+"*)"))
}

func (l *serviceAccountLab) formsConfig(t *testing.T, version int) string {
	t.Helper()
	return l.configuration(t, "forms.tf",
		"__AUTO_NAME__", strconv.Quote(l.prefix+"-automatic"), "__AUTO_SAM__", strconv.Quote(l.sam("automatic")+"$"),
		"__EXPLICIT_NAME__", strconv.Quote(l.prefix+"-explicit"), "__EXPLICIT_SAM__", strconv.Quote(l.sam("explicit")+"$"),
		"__OUTBOUND_NAME__", strconv.Quote(l.prefix+"-outbound"), "__OUTBOUND_SAM__", strconv.Quote(l.sam("outbound")+"$"),
		"__PASSWORD_VERSION__", strconv.Itoa(version), "__FILTER__", strconv.Quote("(name="+l.prefix+"*)"))
}

// TestAccServiceAccountLifecycle verifies actual gMSA CRUD, independent descriptor preservation, discovery, import and drift.
func TestAccServiceAccountLifecycle(t *testing.T) {
	lab := newServiceAccountLab(t)
	firstGroup := lab.createAuxiliary(t, "group", "reader-first", lab.parentGUID)
	secondGroup := lab.createAuxiliary(t, "group", "reader-second", lab.parentGUID)
	firstSID, secondSID := lab.readerSID(t, firstGroup), lab.readerSID(t, secondGroup)
	lab.createAuxiliary(t, "organizationalUnit", "move", lab.parentGUID)
	moveDN := "OU=" + lab.prefix + "-move," + lab.parentDN
	initial := lab.serviceConfig(t, "initial", lab.parentDN, fmt.Sprintf(`
 display_name = "first display"
 service_principal_names = ["HTTP/%s.ad.sdk.test"]
 password_reader_sids = [%q]
 kerberos_encryption_types = ["aes128", "aes256"]
 compound_identity_supported = false
 account_expires = 0`, lab.sam("initial-spn"), firstSID))
	updated := lab.serviceConfig(t, "renamed", moveDN, fmt.Sprintf(`
 display_name = "second display"
 service_principal_names = ["HTTP/%s.ad.sdk.test"]
 password_reader_sids = [%q]
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 account_expires = 9223372036854775807`, lab.sam("updated-spn"), secondSID))
	cleared := lab.serviceConfig(t, "renamed", moveDN, `
 clear_properties = ["displayName"]
 service_principal_names = []
 password_reader_sids = []
 kerberos_encryption_types = ["aes256"]
 compound_identity_supported = true
 account_expires = 9223372036854775807`)
	checks := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{labSecretPlan{secrets: []string{os.Getenv("AD_PASSWORD")}}}}
	var proof readerFixtureProof
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".test", "kind", "group_managed"), resource.TestCheckResourceAttr(ResourceName+".test", "enabled", "false"),
			resource.TestCheckResourceAttrPair("data."+ResourceName+".by_guid", "id", ResourceName+".test", "id"),
			resource.TestCheckResourceAttrPair("data."+ResourceName+".by_dn", "id", ResourceName+".test", "id"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.scoped", "service_accounts.#", "1"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.scoped", "complete", "true"))},
		{Config: initial, PlanOnly: true},
		{
			Config: updated, PreConfig: func() { proof = lab.seedUnrelatedReaderACE(t, firstSID) }, ConfigPlanChecks: checks,
			Check: resource.ComposeAggregateTestCheckFunc(lab.capture, func(*terraform.State) error { return lab.checkReaderPolicy(proof, []string{secondSID}) },
				resource.TestCheckResourceAttr(ResourceName+".test", "parent_dn", moveDN), resource.TestCheckResourceAttr(ResourceName+".test", "display_name", "second display")),
		},
		{Config: updated, PlanOnly: true},
		{ResourceName: ResourceName + ".test", ImportState: true, ImportStateVerify: true},
		{
			ResourceName: ResourceName + ".test", ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity, GenerateConfig: true,
			ImportPlanChecks: resource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
				labSecretPlan{secrets: []string{os.Getenv("AD_PASSWORD")}}, plancheck.ExpectResourceAction(ResourceName+".generated", plancheck.ResourceActionNoop),
			}},
		},
		{
			Query: true, Config: lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"), "__LIMIT__", "10"),
			QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)},
		},
		{Config: cleared, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(
			lab.capture,
			func(*terraform.State) error { return lab.checkReaderPolicy(proof, nil) },
			resource.TestCheckNoResourceAttr(ResourceName+".test", "display_name"),
			resource.TestCheckResourceAttr(
				ResourceName+".test",
				"password_reader_sids.#",
				"0",
			),
			resource.TestCheckResourceAttr(ResourceName+".test", "service_principal_names.#", "0"),
		)},
		{Config: cleared, PlanOnly: true},
		{
			Config: cleared, PreConfig: func() { lab.driftOwnedServiceAccount(t, firstSID) }, ConfigPlanChecks: checks,
			Check: resource.ComposeAggregateTestCheckFunc(lab.capture, func(*terraform.State) error { return lab.checkReaderPolicy(proof, nil) },
				resource.TestCheckNoResourceAttr(ResourceName+".test", "display_name"), resource.TestCheckResourceAttr(ResourceName+".test", "service_principal_names.#", "0")),
		},
		{Config: cleared, PlanOnly: true},
	}})
}

// TestAccServiceAccountCreationForms verifies automatic/explicit sMSA passwords, outbound gMSA, replacement and real paging/limits without secret retrieval.
func TestAccServiceAccountCreationForms(t *testing.T) {
	lab := newServiceAccountLab(t)
	lab.allowReplacement = true
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("fixture-only ephemeral password generation failed")
	}
	defer clear(random)
	lab.secret = "Aa1!" + base64.RawStdEncoding.EncodeToString(random)
	t.Setenv("TF_VAR_service_account_password", lab.secret)
	initial, replaced := lab.formsConfig(t, 1), lab.formsConfig(t, 2)
	planChecks := []plancheck.PlanCheck{labSecretPlan{secrets: []string{os.Getenv("AD_PASSWORD"), lab.secret}}}
	checks := resource.ConfigPlanChecks{PreApply: planChecks}
	replacementChecks := resource.ConfigPlanChecks{PreApply: append([]plancheck.PlanCheck{
		plancheck.ExpectResourceAction(ResourceName+".explicit", plancheck.ResourceActionDestroyBeforeCreate),
		plancheck.ExpectResourceAction(
			ResourceName+".automatic",
			plancheck.ResourceActionNoop,
		),
		plancheck.ExpectResourceAction(ResourceName+".outbound", plancheck.ResourceActionNoop),
	}, planChecks...)}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), CheckDestroy: lab.checkDestroy, Steps: []resource.TestStep{
		{Config: initial, ConfigPlanChecks: checks, Check: resource.ComposeAggregateTestCheckFunc(
			lab.capture,
			resource.TestCheckResourceAttr(
				ResourceName+".automatic",
				"password_wo_version",
				"0",
			),
			resource.TestCheckResourceAttr(ResourceName+".explicit", "password_wo_version", "1"),
			resource.TestCheckResourceAttr(ResourceName+".outbound", "outbound_authentication_only", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.scoped", "service_accounts.#", "3"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.scoped", "complete", "true"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.bounded", "service_accounts.#", "1"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.bounded", "complete", "false"),
			resource.TestCheckResourceAttr("data.activedirectory_service_accounts.bounded", "limit_reached", "true"),
		)},
		{Config: initial, PlanOnly: true},
		{
			Query: true, Config: lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"), "__LIMIT__", "10"),
			QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 3)},
		},
		{
			Query: true, Config: lab.configuration(t, "list-resource.tfquery.hcl", "__FILTER__", strconv.Quote("(name="+lab.prefix+"*)"), "__LIMIT__", "1"),
			QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength(ResourceName+".owned", 1)},
		},
		{Config: replaced, ConfigPlanChecks: replacementChecks, Check: resource.ComposeAggregateTestCheckFunc(lab.capture,
			resource.TestCheckResourceAttr(ResourceName+".explicit", "password_wo_version", "2"))},
		{Config: replaced, PlanOnly: true},
	}})
}
