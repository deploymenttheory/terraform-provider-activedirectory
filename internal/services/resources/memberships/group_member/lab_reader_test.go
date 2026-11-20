// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// reader independently qualifies the ordinary actor without changing its privileges or the selected protection.
func (l *groupMemberLab) reader(t *testing.T) client.Memberships {
	t.Helper()
	username, secret := os.Getenv("TF_VAR_reader_upn"), os.Getenv("TF_VAR_reader_password")
	if username == "" || secret == "" {
		t.Fatal("independently prepared ordinary actor credentials are required")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || !roots.AppendCertsFromPEM([]byte(os.Getenv("AD_CA_CERT_PEM"))) {
		t.Fatal("ordinary actor verified CA trust is required")
	}
	password := []byte(secret)
	defer clear(password)
	bundle, err := client.Open(t.Context(), config.Config{
		Backend: config.BackendPortable, Transport: config.TransportLDAPS, Authentication: config.AuthenticationSimple,
		Endpoint: config.Endpoint{Host: l.target.Host}, Partition: l.target.Partition, Timeout: 30 * time.Second,
		Credentials: &config.Credentials{Username: username, Password: password}, TLS: &config.TLSOptions{RootCAs: roots},
	})
	if err != nil {
		t.Fatal("ordinary actor protected qualification failed")
	}
	t.Cleanup(func() {
		if err := bundle.Close(); err != nil {
			t.Error("ordinary actor session close failed")
		}
	})
	service, err := bundle.Memberships()
	if err != nil {
		t.Fatal("ordinary actor direct-membership service is unavailable")
	}
	return service
}

// hiddenFixture verifies the native permission envelope and prevents inaccessible values from becoming false absence proof.
func (l *groupMemberLab) hiddenFixture(t *testing.T) (labSubject, labSubject, bool) {
	t.Helper()
	group := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_GROUP_GUID"), Kind: "group", External: true}
	member := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_MEMBER_GUID"), Kind: "group", External: true}
	if err := l.assertLink(group, member, true); err != nil {
		t.Fatal("independent native hidden-group fixtures did not verify")
	}
	service := l.reader(t)
	raw, err := service.GroupMemberLinks(t.Context(), l.identity(group), memberships.GroupMemberLinksOptions{MaxValues: 10})
	if err != nil || !raw.Complete || raw.AttributeReturned || len(raw.Members) != 0 {
		t.Fatal("ordinary actor did not observe the independently proven nonempty group with member values hidden")
	}
	edge, readErr := common.Read(t.Context(), service, l.identity(group), l.identity(member))
	if edge.Confirmed && !edge.Present {
		t.Fatal("a hidden existing direct link became confirmed absence")
	}
	collection, collectionErr := common.Discover(t.Context(), service, l.identity(group), 10)
	if collection.Complete || collectionErr == nil || len(collection.Edges) != 0 {
		t.Fatal("a hidden nonempty group became a complete empty collection")
	}
	return group, member, readErr == nil && edge.Confirmed && edge.Present
}
