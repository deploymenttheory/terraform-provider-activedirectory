// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"crypto/x509"
	"embed"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_account"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/service_accounts/service_accounts"
	discovery "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/service_accounts/service_account"
)

//go:embed tests/terraform/acceptance/*
var labConfigurations embed.FS

var errLabOwnership = errors.New("owned service account lab boundary did not verify")

type serviceAccountLab struct {
	target           acceptance.Target
	parentDN         string
	parentGUID       string
	prefix           string
	marker           string
	mu               sync.Mutex
	observer         *activedirectory.Client
	guids            map[string]bool
	childGUID        string
	auxiliaries      map[string]ownedAuxiliary
	lastGUID         map[string]string
	allowReplacement bool
	secret           string
}

// newServiceAccountLab validates the externally owned root and creates no directory fixture by itself.
func newServiceAccountLab(t *testing.T) *serviceAccountLab {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" || os.Getenv("TF_ACC_AD_LAB") != "1" {
		t.Skip("explicit owned-lab acceptance is disabled")
	}
	target := acceptance.PreCheck(t)
	parent := os.Getenv("AD_ACCEPTANCE_PARENT_DN")
	guid, err := shared.ParseGUID(os.Getenv("AD_ACCEPTANCE_PARENT_GUID"))
	if err != nil || guid == (shared.GUID{}) || !strings.EqualFold(parent, "OU=sdk-portable-"+target.RunID+","+target.Partition) {
		t.Fatal("an independently created marker-owned parent OU and its GUID are required")
	}
	prefix, err := acceptance.FixtureName(target.RunID, runtime.GOOS+"/"+t.Name())
	if err != nil || len(prefix) > 54 {
		t.Fatal("choose a shorter run identity for bounded service account fixture names")
	}
	lab := &serviceAccountLab{
		target: target, parentDN: parent, parentGUID: guid.String(), prefix: prefix,
		marker: "tfad-owned:" + target.RunID + ":" + runtime.GOOS, guids: map[string]bool{}, auxiliaries: map[string]ownedAuxiliary{}, lastGUID: map[string]string{},
	}
	t.Setenv(resource.EnvTfAccProviderNamespace, "deploymenttheory")
	t.Chdir(t.TempDir())
	t.Cleanup(func() {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer != nil {
			if err := lab.observer.Close(); err != nil {
				t.Error("close independent lab observer")
			}
		}
	})
	lab.initializeObserver(t)
	acceptance.RegisterCleanup(t, "service_accounts", func(ctx context.Context) (bool, error) {
		lab.mu.Lock()
		defer lab.mu.Unlock()
		if lab.observer == nil {
			return false, errLabOwnership
		}
		result, err := lab.observer.Directory.Search(ctx, shared.SearchOptions{
			Base: parent, Scope: shared.ScopeSubtree, Filter: "(name=" + prefix + "*)", PageSize: 10, ResultLimit: 20,
		})
		if err != nil || !result.Complete || result.LimitReached || len(result.Objects) != 0 {
			return false, errLabOwnership
		}
		for id := range lab.guids {
			target, parseErr := shared.IdentityByGUID(id)
			if parseErr != nil {
				return false, errLabOwnership
			}
			observed, readErr := lab.observer.Directory.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{})
			if !lifecycleAbsent(observed, readErr) {
				return false, errLabOwnership
			}
		}
		for id := range lab.auxiliaries {
			target, parseErr := shared.IdentityByGUID(id)
			if parseErr != nil {
				return false, errLabOwnership
			}
			observed, readErr := lab.observer.Directory.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{})
			if !lifecycleAbsent(observed, readErr) {
				return false, errLabOwnership
			}
		}
		return true, nil
	})
	return lab
}

// factories registers actual production resource/data/list construction on separately authenticated SDK sessions.
func (l *serviceAccountLab) factories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(t, providercore.Registry{
		Resources:     []func() frameworkresource.Resource{NewServiceAccountResource},
		DataSources:   []func() datasource.DataSource{single.NewServiceAccountDataSource, plural.NewServiceAccountsDataSource},
		ListResources: []func() list.ListResource{discovery.NewServiceAccountListResource},
	}, client.Open)
}

// initializeObserver opens a separate verified portable LDAPS session before auxiliary setup and checks the externally owned root.
func (l *serviceAccountLab) initializeObserver(t *testing.T) {
	t.Helper()
	roots, err := x509.SystemCertPool()
	if err != nil || !roots.AppendCertsFromPEM([]byte(os.Getenv("AD_CA_CERT_PEM"))) {
		t.Fatal("independent observer requires verified fixture CA trust")
	}
	password := []byte(os.Getenv("AD_PASSWORD"))
	defer clear(password)
	options := config.Config{
		Backend: config.BackendPortable, Transport: config.TransportLDAPS, Authentication: config.AuthenticationSimple,
		Endpoint: config.Endpoint{Host: l.target.Host}, Partition: l.target.Partition, Timeout: 30 * time.Second,
		Credentials: &config.Credentials{Username: os.Getenv("AD_USERNAME"), Password: password}, TLS: &config.TLSOptions{RootCAs: roots},
	}
	observer, err := activedirectory.Dial(t.Context(), options)
	if err != nil {
		t.Fatal("independent observer connection failed")
	}
	l.observer = observer
	protection, err := observer.Protection(t.Context())
	if err != nil || !protection.Complete || !protection.TLSVerified || protection.CipherBits < 128 || protection.Backend != config.BackendPortable ||
		protection.Transport != config.TransportLDAPS || protection.Authentication != config.AuthenticationSimple {
		t.Fatal("independent observer protection did not verify")
	}
	id, _ := shared.IdentityByGUID(l.parentGUID)
	parent, err := observer.OrganizationalUnits.OrganizationalUnit(t.Context(), id, shared.ResolveIdentityOptions{})
	if err != nil || !parent.Observation.Complete || parent.Unit == nil || parent.Unit.Description == nil || *parent.Unit.Description != "sdk-portable-parent:"+l.target.RunID ||
		!strings.EqualFold(parent.Unit.DistinguishedName, l.parentDN) {
		t.Fatal("external fixture root did not independently verify")
	}
}

func (l *serviceAccountLab) configuration(t *testing.T, name string, replacements ...string) string {
	t.Helper()
	content, err := labConfigurations.ReadFile("tests/terraform/acceptance/" + name)
	if err != nil {
		t.Fatal("embedded acceptance configuration is missing")
	}
	pairs := make([]string, 0, 6+len(replacements))
	pairs = append(pairs, "__PREFIX__", strconv.Quote(l.prefix), "__MARKER__", strconv.Quote(l.marker), "__ROOT_DN__", strconv.Quote(l.parentDN))
	pairs = append(pairs, replacements...)
	return strings.NewReplacer(pairs...).Replace(string(content))
}

// capture verifies GUID stability and records every Terraform-owned object for independent teardown checks.
func (l *serviceAccountLab) capture(state *terraform.State) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, module := range state.Modules {
		for address, object := range module.Resources {
			if object.Type != ResourceName || object.Primary == nil {
				continue
			}
			id := object.Primary.ID
			guid, err := shared.ParseGUID(id)
			if err != nil || guid == (shared.GUID{}) {
				return errLabOwnership
			}
			if _, err := l.ownedObject(context.Background(), guid.String()); err != nil {
				return errLabOwnership
			}
			for _, value := range object.Primary.Attributes {
				for _, secret := range []string{os.Getenv("AD_PASSWORD"), l.secret} {
					if secret != "" && strings.Contains(value, secret) {
						return errLabOwnership
					}
				}
			}
			if old := l.lastGUID[address]; old != "" && old != guid.String() {
				if !l.allowReplacement {
					return errLabOwnership
				}
				target, _ := shared.IdentityByGUID(old)
				observed, readErr := l.observer.Directory.ResolveIdentity(context.Background(), target, shared.ResolveIdentityOptions{})
				if !lifecycleAbsent(observed, readErr) {
					return errLabOwnership
				}
			}
			l.lastGUID[address] = guid.String()
			l.guids[guid.String()] = true
			if address == ResourceName+".test" {
				if l.childGUID != "" && l.childGUID != guid.String() {
					return errLabOwnership
				}
				l.childGUID = guid.String()
			}
		}
	}
	return nil
}

// checkDestroy verifies that every tracked object GUID is absent using a separately authenticated observer.
func (l *serviceAccountLab) checkDestroy(_ *terraform.State) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.observer == nil {
		return errLabOwnership
	}
	for id := range l.guids {
		target, err := shared.IdentityByGUID(id)
		if err != nil {
			return errLabOwnership
		}
		result, err := l.observer.Directory.ResolveIdentity(context.Background(), target, shared.ResolveIdentityOptions{})
		if !lifecycleAbsent(result, err) {
			return errLabOwnership
		}
	}
	return nil
}

// lifecycleAbsent delegates the same pure-not-found decision used by production state removal.
func lifecycleAbsent(result shared.IdentityResult, err error) bool {
	return lifecycle.ConfirmedAbsent(result, err)
}
