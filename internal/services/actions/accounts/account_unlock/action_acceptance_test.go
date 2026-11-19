// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package accountUnlock

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
)

//go:embed tests/terraform/acceptance/action.tf
var labActionConfiguration string

var errLabAccount = errors.New("owned account fixture did not independently verify")

type accountLab struct {
	mu       sync.Mutex
	observer *activedirectory.Client
	identity shared.Identity
	dn       string
	marker   string
	flags    shared.Flags32
	initial  shared.FileTime
}

// TestAccAccountUnlock verifies real Terraform invocation, refusal, retained disabled status and explicit recovery on the selected DC.
func TestAccAccountUnlock(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" || os.Getenv("TF_ACC_AD_LAB") != "1" {
		t.Skip("explicit owned-lab acceptance is disabled")
	}
	target := acceptance.PreCheck(t)
	identity, err := shared.IdentityByGUID(os.Getenv("AD_ACCEPTANCE_ACCOUNT_GUID"))
	parent := os.Getenv("AD_ACCEPTANCE_PARENT_DN")
	dn := os.Getenv("AD_ACCEPTANCE_ACCOUNT_DN")
	expectedDN := "CN=tfad-" + target.RunID + "-unlock-" + runtime.GOOS + "," + parent
	if err != nil || !strings.EqualFold(parent, "OU=sdk-portable-"+target.RunID+","+target.Partition) || !strings.EqualFold(dn, expectedDN) ||
		os.Getenv("TF_VAR_reader_upn") == "" || os.Getenv("TF_VAR_reader_password") == "" {
		t.Fatal("independently prepared marker-owned disabled account and ordinary reader credentials are required")
	}
	lab := &accountLab{identity: identity, dn: dn, marker: "tfad-unlock:" + target.RunID}
	t.Setenv(resource.EnvTfAccProviderNamespace, "deploymenttheory")
	t.Chdir(t.TempDir())
	t.Cleanup(func() {
		if lab.observer != nil {
			if err := lab.observer.Close(); err != nil {
				t.Error("close independent account observer")
			}
		}
	})
	acceptance.RegisterCleanup(t, "account unlock", lab.cleanup)
	factory := func(ctx context.Context, options config.Config) (*client.Bundle, error) {
		bundle, openErr := client.Open(ctx, options)
		if openErr != nil {
			return bundle, fmt.Errorf("open account action provider: %w", openErr)
		}
		if options.Credentials != nil && options.Credentials.Username == os.Getenv("TF_VAR_reader_upn") {
			reader, readerErr := activedirectory.Dial(ctx, options)
			if readerErr != nil {
				return nil, errors.Join(errLabAccount, bundle.Close())
			}
			read, readErr := reader.Accounts.AccountControl(ctx, identity, shared.ResolveIdentityOptions{})
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || !read.Observation.Complete || read.Account == nil || read.Account.LockoutTime == nil || read.Account.StoredFlags == nil {
				return nil, errors.Join(errLabAccount, bundle.Close())
			}
		}
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil && options.Credentials != nil && options.Credentials.Username == os.Getenv("AD_USERNAME") {
			observer, observerErr := activedirectory.Dial(ctx, options)
			if observerErr != nil {
				return nil, errors.Join(errLabAccount, bundle.Close())
			}
			observed, readErr := observer.Accounts.AccountControl(ctx, identity, shared.ResolveIdentityOptions{Properties: []string{"description"}})
			if readErr != nil || !observed.Observation.Complete || observed.Account == nil || observed.Account.Enabled == nil || *observed.Account.Enabled ||
				observed.Account.LockoutTime == nil || *observed.Account.LockoutTime == 0 || observed.Account.StoredFlags == nil ||
				!strings.EqualFold(observed.Account.DistinguishedName, dn) || !lab.owned(observed.Account.Raw) {
				return nil, errors.Join(errLabAccount, observer.Close(), bundle.Close())
			}
			lab.observer, lab.flags, lab.initial = observer, *observed.Account.StoredFlags, *observed.Account.LockoutTime
		}
		return bundle, nil
	}
	factories := acceptance.ProviderFactories(t, providercore.Registry{Actions: []func() action.Action{NewAccountUnlockAction}}, factory)
	configuration := func(provider, server string) string {
		return strings.NewReplacer("__PROVIDER__", provider, "__GUID__", strconv.Quote(identity.String()), "__SERVER__", strconv.Quote(server)).Replace(labActionConfiguration)
	}
	assertInitial := func() {
		if err := lab.check(t.Context(), false); err != nil {
			t.Fatal("refused unlock changed the owned account")
		}
	}
	admin := configuration("activedirectory", target.Host)
	resource.Test(
		t,
		resource.TestCase{ProtoV6ProviderFactories: factories, CheckDestroy: func(*terraform.State) error { return lab.check(t.Context(), true) }, Steps: []resource.TestStep{
			{Config: configuration("activedirectory", "different.unit.invalid"), ExpectError: regexp.MustCompile("Account unlock server differs")},
			{Config: configuration("activedirectory.reader", target.Host), PreConfig: assertInitial, ExpectError: regexp.MustCompile("Directory mutation did not complete")},
			{Config: admin, PreConfig: assertInitial, Check: func(*terraform.State) error { return lab.check(t.Context(), true) }},
			{Config: admin, PlanOnly: true},
		}},
	)
}

// owned verifies the immutable fixture GUID, expected DN and independently assigned ownership marker.
func (l *accountLab) owned(object shared.Object) bool {
	guidValue, guidErr := object.Values("objectGUID").Single()
	guid, parseErr := guidValue.GUID()
	marker, markerErr := object.Values("description").Single()
	return guidErr == nil && parseErr == nil && guid.String() == l.identity.String() && strings.EqualFold(object.DistinguishedName, l.dn) &&
		markerErr == nil && string(marker.Bytes()) == l.marker
}

// check compares the selected DC's actual account values without inferring lockout eligibility from a timestamp.
func (l *accountLab) check(ctx context.Context, unlocked bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.observer == nil {
		return errLabAccount
	}
	result, err := l.observer.Accounts.AccountControl(ctx, l.identity, shared.ResolveIdentityOptions{Properties: []string{"description"}})
	if err != nil || !result.Observation.Complete || result.Account == nil || !l.owned(result.Account.Raw) || result.Account.Enabled == nil || *result.Account.Enabled ||
		result.Account.StoredFlags == nil || *result.Account.StoredFlags != l.flags || result.Account.LockoutTime == nil {
		return errLabAccount
	}
	expected := l.initial
	if unlocked {
		expected = 0
	}
	if *result.Account.LockoutTime != expected {
		return errLabAccount
	}
	return nil
}

// cleanup removes only the exact owned fixture after the Terraform process has stopped using it.
func (l *accountLab) cleanup(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.observer == nil {
		return false, errLabAccount
	}
	observed, err := l.observer.Directory.ResolveIdentity(ctx, l.identity, shared.ResolveIdentityOptions{Properties: []string{"description"}})
	if lifecycle.ConfirmedAbsent(observed, err) {
		return true, nil
	}
	if err != nil || !observed.Complete || observed.Object == nil || !l.owned(*observed.Object) {
		return false, errLabAccount
	}
	deleted, err := l.observer.Directory.DeleteObject(ctx, shared.DeleteObjectRequest{Identity: l.identity, Recursive: false})
	if !lifecycle.ConfirmedDeletion(deleted, err) {
		return false, errLabAccount
	}
	absent, readErr := l.observer.Directory.ResolveIdentity(ctx, l.identity, shared.ResolveIdentityOptions{})
	return lifecycle.ConfirmedAbsent(absent, readErr), nil
}
