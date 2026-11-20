// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package account_settings

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	lookup "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/accounts/account_settings"
)

//go:embed tests/terraform/acceptance/*
var labConfigurations embed.FS

var errLabOwnership = errors.New("owned account-settings lab boundary did not verify")

type labSubject struct{ GUID, Kind string }

type accountSettingsLab struct {
	target                               acceptance.Target
	parentDN, parentGUID, prefix, marker string
	observer                             *activedirectory.Client
	subjects                             map[string]labSubject
	secrets                              []string
}

// newAccountSettingsLab opens an independent protected observer and verifies the externally owned root before fixture effects.
func newAccountSettingsLab(t *testing.T) *accountSettingsLab {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" || os.Getenv("TF_ACC_AD_LAB") != "1" {
		t.Skip("explicit owned-lab acceptance is disabled")
	}
	target := acceptance.PreCheck(t)
	parent := os.Getenv("AD_ACCEPTANCE_PARENT_DN")
	guid, err := shared.ParseGUID(os.Getenv("AD_ACCEPTANCE_PARENT_GUID"))
	if err != nil || guid == (shared.GUID{}) || !strings.EqualFold(parent, "OU=sdk-portable-"+target.RunID+","+target.Partition) {
		t.Fatal("independently owned fixture parent and GUID are required")
	}
	prefix, err := acceptance.FixtureName(target.RunID, runtime.GOOS+"/"+t.Name())
	if err != nil || len(prefix) > 50 {
		t.Fatal("bounded fixture name did not verify")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || !roots.AppendCertsFromPEM([]byte(os.Getenv("AD_CA_CERT_PEM"))) {
		t.Fatal("observer fixture CA trust is required")
	}
	password := []byte(os.Getenv("AD_PASSWORD"))
	defer clear(password)
	observer, err := activedirectory.Dial(t.Context(), config.Config{
		Backend: config.BackendPortable, Transport: config.TransportLDAPS, Authentication: config.AuthenticationSimple,
		Endpoint: config.Endpoint{Host: target.Host}, Partition: target.Partition, Timeout: 30 * time.Second,
		Credentials: &config.Credentials{Username: os.Getenv("AD_USERNAME"), Password: password}, TLS: &config.TLSOptions{RootCAs: roots},
	})
	if err != nil {
		t.Fatal("independent protected observer failed")
	}
	lab := &accountSettingsLab{
		target:     target,
		parentDN:   parent,
		parentGUID: guid.String(),
		prefix:     prefix,
		marker:     "tfad-owned:" + target.RunID + ":" + runtime.GOOS,
		observer:   observer,
		subjects:   map[string]labSubject{},
		secrets:    []string{os.Getenv("AD_PASSWORD"), os.Getenv("TF_VAR_reader_password")},
	}
	t.Cleanup(func() {
		if err := observer.Close(); err != nil {
			t.Error("independent observer close did not verify")
		}
	})
	protection, err := observer.Protection(t.Context())
	if err != nil || !protection.Complete || !protection.TLSVerified || protection.CipherBits < 128 || protection.Backend != config.BackendPortable ||
		protection.Transport != config.TransportLDAPS ||
		protection.Authentication != config.AuthenticationSimple {
		t.Fatal("independent observer protection did not verify")
	}
	id, _ := shared.IdentityByGUID(lab.parentGUID)
	root, err := observer.OrganizationalUnits.OrganizationalUnit(t.Context(), id, shared.ResolveIdentityOptions{})
	if err != nil || !root.Observation.Complete || root.Unit == nil || root.Unit.Description == nil || *root.Unit.Description != "sdk-portable-parent:"+target.RunID ||
		!strings.EqualFold(root.Unit.DistinguishedName, parent) {
		t.Fatal("external root ownership did not verify")
	}
	acceptance.RegisterCleanup(t, "adopted account settings", func(ctx context.Context) (bool, error) {
		if err := lab.retire(ctx); err != nil {
			return false, err
		}
		found, err := observer.Directory.Search(
			ctx,
			shared.SearchOptions{Base: parent, Scope: shared.ScopeSubtree, Filter: "(name=" + prefix + "*)", PageSize: 10, ResultLimit: 20},
		)
		if err != nil || !found.Complete || found.LimitReached || len(found.Objects) != 0 {
			return false, errLabOwnership
		}
		for _, subject := range lab.subjects {
			target, _ := shared.IdentityByGUID(subject.GUID)
			observed, readErr := observer.Directory.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{})
			if !lifecycle.ConfirmedAbsent(observed, readErr) {
				return false, errLabOwnership
			}
		}
		return true, nil
	})
	t.Setenv(resource.EnvTfAccProviderNamespace, "deploymenttheory")
	t.Chdir(t.TempDir())
	return lab
}

// factories registers the real resource and data source on separately authenticated provider sessions.
func (*accountSettingsLab) factories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(
		t,
		core.Registry{Resources: []func() frameworkresource.Resource{NewAccountSettingsResource}, DataSources: []func() datasource.DataSource{lookup.NewAccountSettingsDataSource}},
		client.Open,
	)
}

// configuration substitutes only bounded fixture identities and known nonsecret setting text into embedded HCL.
func (l *accountSettingsLab) configuration(t *testing.T, name string, replacements ...string) string {
	t.Helper()
	data, err := labConfigurations.ReadFile("tests/terraform/acceptance/" + name)
	if err != nil {
		t.Fatal("embedded acceptance configuration is missing")
	}
	return strings.NewReplacer(replacements...).Replace(string(data))
}

// sam keeps each fixture identity within account-name bounds independently of its longer common name.
func (l *accountSettingsLab) sam(kind string) string {
	digest := sha256.Sum256([]byte(l.prefix + "/" + kind))
	return fmt.Sprintf("tfa%x", digest[:6])
}

// ownedObject verifies the exact GUID, marker, expected leaf class and bounded root before any fixture mutation or retirement.
func (l *accountSettingsLab) ownedObject(ctx context.Context, subject labSubject) (*shared.Object, error) {
	target, err := shared.IdentityByGUID(subject.GUID)
	if err != nil {
		return nil, errLabOwnership
	}
	observed, err := l.observer.Directory.ResolveIdentity(
		ctx,
		target,
		shared.ResolveIdentityOptions{Properties: []string{"objectGUID", "name", "description", "objectClass", "distinguishedName"}},
	)
	if err != nil || !observed.Complete || observed.Object == nil {
		return nil, errLabOwnership
	}
	object := observed.Object
	guidValue, guidErr := object.Values("objectGUID").Single()
	guid, parseErr := guidValue.GUID()
	nameValue, nameErr := object.Values("name").Single()
	markerValue, markerErr := object.Values("description").Single()
	classes, state := object.Attribute("objectClass")
	if guidErr != nil || parseErr != nil || guid.String() != subject.GUID || nameErr != nil || markerErr != nil || string(markerValue.Bytes()) != l.marker ||
		!strings.HasPrefix(string(nameValue.Bytes()), l.prefix+"-") || !strings.HasSuffix(strings.ToLower(object.DistinguishedName), ","+strings.ToLower(l.parentDN)) ||
		state != shared.AttributePresent || len(classes) == 0 || !strings.EqualFold(string(classes[len(classes)-1]), subject.Kind) {
		return nil, errLabOwnership
	}
	return object, nil
}

// capture checks every managed state against an independently owned existing account and rejects credential material in state.
func (l *accountSettingsLab) capture(state *terraform.State) error {
	if state == nil {
		return nil
	}
	for _, module := range state.Modules {
		for address, managed := range module.Resources {
			if strings.HasPrefix(address, "data.") {
				continue
			}
			if managed.Type != ResourceName || managed.Primary == nil {
				continue
			}
			id := managed.Primary.ID
			subject, ok := l.subjects[id]
			if !ok {
				return errLabOwnership
			}
			if _, err := l.ownedObject(context.Background(), subject); err != nil {
				return errLabOwnership
			}
			if managed.Primary.Attributes["account_guid"] != id {
				return errLabOwnership
			}
			for _, value := range managed.Primary.Attributes {
				for _, secret := range l.secrets {
					if secret != "" && strings.Contains(value, secret) {
						return errLabOwnership
					}
				}
			}
		}
	}
	return nil
}

// checkDestroy proves Terraform relinquished management while all independently provisioned accounts remain live.
func (l *accountSettingsLab) checkDestroy(*terraform.State) error {
	for _, subject := range l.subjects {
		if _, err := l.ownedObject(context.Background(), subject); err != nil {
			return errLabOwnership
		}
	}
	return nil
}

// identity returns one already-verified fixture GUID without using a name as mutation identity.
func (l *accountSettingsLab) identity(subject labSubject) shared.Identity {
	target, _ := shared.IdentityByGUID(subject.GUID)
	return target
}

// quotedGUID formats only a known nonsecret identifier for its Terraform configuration.
func quotedGUID(subject labSubject) string { return strconv.Quote(subject.GUID) }
