// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"bytes"
	"context"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/go-sdk-activedirectory/conditions"
)

type readerFixtureProof struct {
	ace   []byte
	owner []byte
	group []byte
	sacl  conditions.ACL
}

// readerSID returns only an independently marker/GUID-verified owned group's SID.
func (l *serviceAccountLab) readerSID(t *testing.T, guid string) string {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	object, err := l.ownedObject(t.Context(), guid)
	if err != nil {
		t.Fatal("owned reader group did not verify")
	}
	value, err := object.Values("objectSid").Single()
	if err != nil {
		t.Fatal("owned reader SID is unavailable")
	}
	sid, err := value.SID()
	if err != nil {
		t.Fatal("owned reader SID is invalid")
	}
	return sid.String()
}

// seedUnrelatedReaderACE writes one unrelated WP-only fixture ACE below the verified subject boundary, without replay.
func (l *serviceAccountLab) seedUnrelatedReaderACE(t *testing.T, sid string) readerFixtureProof {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	object, err := l.ownedObject(t.Context(), l.childGUID)
	if err != nil {
		t.Fatal("owned subject did not verify before descriptor fixture setup")
	}
	value, err := object.Values("msDS-GroupMSAMembership").Single()
	if err != nil {
		t.Fatal("reader descriptor is unavailable")
	}
	before, err := conditions.ParseDescriptor(value.Bytes())
	if err != nil {
		t.Fatal("reader descriptor could not be decoded")
	}
	seed, err := conditions.ParseSDDL("O:SYG:SYD:(A;;WP;;;" + sid + ")")
	if err != nil || len(seed.DACL().ACEs) != 1 {
		t.Fatal("unrelated fixture ACE could not be encoded")
	}
	acl := before.DACL()
	if !acl.Present || acl.Null {
		t.Fatal("reader fixture requires a concrete visible DACL")
	}
	ace := seed.DACL().ACEs[0]
	acl.ACEs = append(acl.ACEs, ace)
	updated, err := before.WithDACL(acl)
	if err != nil {
		t.Fatal("unrelated fixture ACE could not be appended")
	}
	encoded, err := updated.MarshalBinary()
	if err != nil {
		t.Fatal("reader fixture descriptor could not be encoded")
	}
	target, _ := shared.IdentityByGUID(l.childGUID)
	result, err := l.observer.Directory.UpdateObject(t.Context(), shared.UpdateObjectRequest{Identity: target, Changes: []shared.AttributeChange{
		{Attribute: "msDS-GroupMSAMembership", Kind: shared.AttributeSet, Values: []shared.RawValue{shared.NewRawValue(encoded)}},
	}})
	if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified || result.TargetGUID == nil || result.TargetGUID.String() != l.childGUID {
		t.Fatal("unrelated descriptor fixture write did not verify; no replay attempted")
	}
	return readerFixtureProof{ace: bytes.Clone(ace), owner: before.OwnerSID(), group: before.GroupSID(), sacl: before.SACL()}
}

// checkReaderPolicy checks actual grants and byte-preserved unrelated descriptor components with a separate authenticated observer.
func (l *serviceAccountLab) checkReaderPolicy(proof readerFixtureProof, want []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	object, err := l.ownedObject(context.Background(), l.childGUID)
	if err != nil {
		return errLabOwnership
	}
	value, err := object.Values("msDS-GroupMSAMembership").Single()
	if err != nil {
		return errLabOwnership
	}
	descriptor, err := conditions.ParseDescriptor(value.Bytes())
	if err != nil || !bytes.Equal(descriptor.OwnerSID(), proof.owner) || !bytes.Equal(descriptor.GroupSID(), proof.group) {
		return errLabOwnership
	}
	actualSACL := descriptor.SACL()
	if actualSACL.Present != proof.sacl.Present || actualSACL.Null != proof.sacl.Null || actualSACL.Revision != proof.sacl.Revision ||
		len(actualSACL.ACEs) != len(proof.sacl.ACEs) {
		return errLabOwnership
	}
	for i, ace := range actualSACL.ACEs {
		if !bytes.Equal(ace, proof.sacl.ACEs[i]) {
			return errLabOwnership
		}
	}
	preserved := false
	for _, ace := range descriptor.DACL().ACEs {
		preserved = preserved || bytes.Equal(ace, proof.ace)
	}
	if !preserved {
		return errLabOwnership
	}
	target, _ := shared.IdentityByGUID(l.childGUID)
	result, err := l.observer.ServiceAccounts.ServiceAccount(context.Background(), target, shared.ResolveIdentityOptions{})
	if err != nil || !result.Observation.Complete || result.Account == nil || !result.Account.PasswordReaderDescriptorReturned {
		return errLabOwnership
	}
	actual := []string{}
	for _, sid := range result.Account.PasswordReaders {
		actual = append(actual, sid.String())
	}
	if !sameStringSet(actual, want) {
		return errLabOwnership
	}
	return nil
}

// driftOwnedServiceAccount changes only the verified fixture subject, retaining the permanent description marker.
func (l *serviceAccountLab) driftOwnedServiceAccount(t *testing.T, reader string) {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.ownedObject(t.Context(), l.childGUID); err != nil {
		t.Fatal("subject ownership failed before independent drift")
	}
	target, _ := shared.IdentityByGUID(l.childGUID)
	principal, err := shared.IdentityBySID(reader)
	if err != nil {
		t.Fatal("owned reader SID is invalid")
	}
	readers := []shared.Identity{principal}
	display := "independent fixture drift"
	spns := []string{"HTTP/" + l.sam("drift") + ".ad.sdk.test"}
	result, err := l.observer.ServiceAccounts.UpdateServiceAccount(t.Context(), service_accounts.UpdateServiceAccountRequest{
		Identity: target, Properties: service_accounts.ServiceAccountProperties{DisplayName: &display, ServicePrincipalNames: &spns}, PasswordReaders: &readers,
	})
	if err != nil || !result.Mutation.Verified || result.Mutation.TargetGUID == nil || result.Mutation.TargetGUID.String() != l.childGUID {
		t.Fatal("independent service account drift did not verify")
	}
}
