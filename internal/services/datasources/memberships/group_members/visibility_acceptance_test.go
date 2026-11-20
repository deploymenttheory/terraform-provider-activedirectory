// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/memberships"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
)

// TestAccGroupMembersVisibility proves a native nonempty hidden group cannot become a complete empty direct or recursive query.
func TestAccGroupMembersVisibility(t *testing.T) {
	lab := newGroupMembersLab(t)
	group := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_GROUP_GUID"), Kind: "group", External: true}
	member := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_MEMBER_GUID"), Kind: "group", External: true}
	if err := lab.assertLink(group, member, true); err != nil {
		t.Fatal("independently prepared hidden group link did not verify")
	}
	reader := lab.reader(t)
	options := memberships.GroupMembersOptions{Limits: memberships.MembershipLimits{MaxObjects: 100, MaxGroups: 10, MaxRanges: 10, MaxDepth: 4, MaxBytes: 1 << 20}}
	observed, err := reader.GroupMembers(t.Context(), lab.identity(group), options)
	if err != nil || len(observed.Groups) != 1 || observed.Groups[0].Links.AttributeReturned || len(observed.Members) != 0 {
		t.Fatal("ordinary actor did not independently observe the native member-property omission")
	}
	configuration := func(recursive string) string {
		return lab.configuration(t, "visibility.tf", "__GROUP_GUID__", group.GUID, "__RECURSIVE__", recursive)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: lab.factories(t), Steps: []resource.TestStep{
		{Config: configuration("false"), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".hidden", "members.#", "0"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".hidden", "complete", "false"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".hidden", "visibility_known", "false"),
		)},
		{Config: configuration("true"), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data."+DataSourceName+".hidden", "members.#", "0"),
			resource.TestCheckResourceAttr("data."+DataSourceName+".hidden", "complete", "false"),
		)},
	}})
	if err := lab.assertLink(group, member, true); err != nil {
		t.Fatal("read-only hidden discovery changed the native link")
	}
}

// reader independently qualifies the ordinary actor through verified TLS without altering its rights or forwarding credentials.
func (l *groupMembersLab) reader(t *testing.T) client.GroupMembershipQueries {
	t.Helper()
	username, secret := os.Getenv("TF_VAR_reader_upn"), os.Getenv("TF_VAR_reader_password")
	if username == "" || secret == "" {
		t.Fatal("independently prepared ordinary actor credentials are required")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || !roots.AppendCertsFromPEM([]byte(os.Getenv("AD_CA_CERT_PEM"))) {
		t.Fatal("ordinary actor CA trust did not verify")
	}
	password := []byte(secret)
	defer clear(password)
	bundle, err := client.Open(t.Context(), config.Config{
		Backend: config.BackendPortable, Transport: config.TransportLDAPS, Authentication: config.AuthenticationSimple,
		Endpoint: config.Endpoint{Host: l.target.Host}, Partition: l.target.Partition, Timeout: 30 * time.Second,
		Credentials: &config.Credentials{Username: username, Password: password}, TLS: &config.TLSOptions{RootCAs: roots},
	})
	if err != nil {
		t.Fatal("ordinary actor protection qualification failed")
	}
	t.Cleanup(func() {
		if err := bundle.Close(); err != nil {
			t.Error("ordinary actor session retirement did not verify")
		}
	})
	service, err := bundle.GroupMembershipQueries()
	if err != nil {
		t.Fatal("ordinary actor query service is unavailable")
	}
	return service
}
