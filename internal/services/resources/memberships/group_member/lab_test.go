// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
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
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	core "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
	lookup "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_member"
	plural "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/datasources/memberships/group_member_relationships"
	discovery "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/list-resources/memberships/group_member"
)

//go:embed tests/terraform/acceptance/*
var labConfigurations embed.FS

var errLabOwnership = errors.New("owned group-member lab boundary did not verify")

type labSubject struct {
	GUID, Kind string
	External   bool
}

type groupMemberLab struct {
	target                               acceptance.Target
	parentDN, parentGUID, prefix, marker string
	observer                             *activedirectory.Client
	subjects                             map[string]labSubject
	secrets                              []string
}

// newGroupMemberLab opens an independent protected observer and verifies the externally owned root before fixture effects.
func newGroupMemberLab(t *testing.T) *groupMemberLab {
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
	lab := &groupMemberLab{
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
	acceptance.RegisterCleanup(t, "direct group memberships", func(ctx context.Context) (bool, error) {
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
func (*groupMemberLab) factories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	return acceptance.ProviderFactories(
		t,
		core.Registry{
			Resources:     []func() frameworkresource.Resource{NewGroupMemberResource},
			DataSources:   []func() datasource.DataSource{lookup.NewGroupMemberDataSource, plural.NewGroupMemberRelationshipsDataSource},
			ListResources: []func() list.ListResource{discovery.NewGroupMemberListResource},
		},
		client.Open,
	)
}

// configuration substitutes only bounded fixture identities and known nonsecret setting text into embedded HCL.
func (l *groupMemberLab) configuration(t *testing.T, name string, replacements ...string) string {
	t.Helper()
	data, err := labConfigurations.ReadFile("tests/terraform/acceptance/" + name)
	if err != nil {
		t.Fatal("embedded acceptance configuration is missing")
	}
	return strings.NewReplacer(replacements...).Replace(string(data))
}

// sam keeps each fixture identity within account-name bounds independently of its longer common name.
func (l *groupMemberLab) sam(kind string) string {
	digest := sha256.Sum256([]byte(l.prefix + "/" + kind))
	return fmt.Sprintf("tfa%x", digest[:6])
}

// ownedObject verifies the exact GUID, marker, expected leaf class and bounded root before any fixture mutation or retirement.
func (l *groupMemberLab) ownedObject(ctx context.Context, subject labSubject) (*shared.Object, error) {
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
	prefix := l.prefix + "-"
	if subject.External {
		prefix = "tfad-" + l.target.RunID + "-hidden-"
	}
	if guidErr != nil || parseErr != nil || guid.String() != subject.GUID || nameErr != nil || markerErr != nil || string(markerValue.Bytes()) != l.marker ||
		!strings.HasPrefix(string(nameValue.Bytes()), prefix) || !strings.HasSuffix(strings.ToLower(object.DistinguishedName), ","+strings.ToLower(l.parentDN)) ||
		state != shared.AttributePresent || len(classes) == 0 || !strings.EqualFold(string(classes[len(classes)-1]), subject.Kind) {
		return nil, errLabOwnership
	}
	return object, nil
}

// capture checks each managed GUID pair against independent owned objects and rejects credential material in state.
func (l *groupMemberLab) capture(state *terraform.State) error {
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
			group, member, diagnostics := identity.ParseRelationshipID(managed.Primary.ID)
			if diagnostics.HasError() {
				return errLabOwnership
			}
			for field, id := range map[string]string{"group_guid": group, "member_guid": member} {
				subject, ok := l.subjects[id]
				if !ok || managed.Primary.Attributes[field] != id {
					return errLabOwnership
				}
				if _, err := l.ownedObject(context.Background(), subject); err != nil {
					return errLabOwnership
				}
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

// checkDestroy proves link deletion retained all independently provisioned endpoint objects.
func (l *groupMemberLab) checkDestroy(*terraform.State) error {
	for _, subject := range l.subjects {
		if _, err := l.ownedObject(context.Background(), subject); err != nil {
			return errLabOwnership
		}
	}
	return nil
}

// identity returns one already-verified fixture GUID without using a name as mutation identity.
func (l *groupMemberLab) identity(subject labSubject) shared.Identity {
	target, _ := shared.IdentityByGUID(subject.GUID)
	return target
}
