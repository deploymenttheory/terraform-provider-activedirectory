// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer_service_account

import (
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/config"
	sdkaccounts "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/client"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_account_links"
)

// reader independently qualifies the ordinary actor without changing its privileges or the selected protection.
func (l *computerServiceAccountLab) reader(t *testing.T) client.ComputerServiceAccountLinks {
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
	service, err := bundle.ComputerServiceAccountLinks()
	if err != nil {
		t.Fatal("ordinary actor direct-association service is unavailable")
	}
	return service
}

// hiddenFixture verifies the native permission envelope and prevents inaccessible values from becoming false absence proof.
func (l *computerServiceAccountLab) hiddenFixture(t *testing.T) (labSubject, labSubject, bool) {
	t.Helper()
	computer := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_COMPUTER_GUID"), Kind: "computer", External: true}
	account := labSubject{GUID: os.Getenv("AD_ACCEPTANCE_HIDDEN_SERVICE_ACCOUNT_GUID"), Kind: "msDS-ManagedServiceAccount", External: true}
	if err := l.assertLink(computer, account, true); err != nil {
		t.Fatal("independent native hidden-computer fixtures did not verify")
	}
	service := l.reader(t)
	raw, err := service.ComputerServiceAccounts(t.Context(), l.identity(computer), sdkaccounts.ComputerServiceAccountsOptions{MaxValues: 10})
	if err != nil || !raw.Complete || raw.AttributeReturned || len(raw.Links) != 0 {
		t.Fatal("ordinary actor did not observe the independently proven nonempty computer with account values hidden")
	}
	edge, readErr := common.Read(t.Context(), service, l.identity(computer), l.identity(account))
	if edge.Confirmed && !edge.Present {
		t.Fatal("a hidden existing direct link became confirmed absence")
	}
	collection, collectionErr := common.Discover(t.Context(), service, l.identity(computer), 10)
	if collection.Complete || collectionErr == nil || len(collection.Edges) != 0 {
		t.Fatal("a hidden nonempty computer became a complete empty collection")
	}
	return computer, account, readErr == nil && edge.Confirmed && edge.Present
}
