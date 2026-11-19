// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

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
	single "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/object"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/objects/objects"
	discovery "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/objects/object"
)

//go:embed tests/terraform/acceptance/*
var labConfigurations embed.FS

var errLabOwnership = errors.New("owned object lab boundary did not verify")

type objectLab struct {
	target      acceptance.Target
	parentDN    string
	parentGUID  string
	prefix      string
	marker      string
	mu          sync.Mutex
	observer    *activedirectory.Client
	guids       map[string]bool
	childGUID   string
	auxiliaries map[string]ownedAuxiliary
}

// newObjectLab validates the externally owned root and creates no directory fixture by itself.
func newObjectLab(t *testing.T) *objectLab {
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
		t.Fatal("choose a shorter run identity for bounded object fixture names")
	}
	lab := &objectLab{
		target: target, parentDN: parent, parentGUID: guid.String(), prefix: prefix,
		marker: "tfad-owned:" + target.RunID + ":" + runtime.GOOS, guids: map[string]bool{}, auxiliaries: map[string]ownedAuxiliary{},
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
	acceptance.RegisterCleanup(t, "objects", func(ctx context.Context) (bool, error) {
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

// factories uses actual production SDK construction; the independent observer is a separate real LDAP session.
func (l *objectLab) factories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	factory := func(ctx context.Context, options config.Config) (*client.Bundle, error) {
		bundle, err := client.Open(ctx, options)
		if err != nil {
			return bundle, fmt.Errorf("open lab provider session: %w", err)
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.observer == nil && options.Credentials != nil && options.Credentials.Username == os.Getenv("AD_USERNAME") {
			observer, openErr := activedirectory.Dial(ctx, options)
			if openErr != nil {
				return nil, errors.Join(errLabOwnership, bundle.Close())
			}
			id, _ := shared.IdentityByGUID(l.parentGUID)
			parent, readErr := observer.OrganizationalUnits.OrganizationalUnit(ctx, id, shared.ResolveIdentityOptions{})
			if readErr != nil || !parent.Observation.Complete || parent.Unit == nil || parent.Unit.Description == nil ||
				*parent.Unit.Description != "sdk-portable-parent:"+l.target.RunID || !strings.EqualFold(parent.Unit.DistinguishedName, l.parentDN) {
				return nil, errors.Join(errLabOwnership, observer.Close(), bundle.Close())
			}
			l.observer = observer
		}
		return bundle, nil
	}
	return acceptance.ProviderFactories(t, providercore.Registry{
		Resources:     []func() frameworkresource.Resource{NewObjectResource},
		DataSources:   []func() datasource.DataSource{single.NewObjectDataSource, plural.NewObjectsDataSource},
		ListResources: []func() list.ListResource{discovery.NewObjectListResource},
	}, factory)
}

func (l *objectLab) configuration(t *testing.T, name string, replacements ...string) string {
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
func (l *objectLab) capture(state *terraform.State) error {
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
func (l *objectLab) checkDestroy(_ *terraform.State) error {
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
