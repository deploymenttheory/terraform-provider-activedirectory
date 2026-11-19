// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestDeletionProofRejectsOtherGUIDAndMixedAbsence protects state when deletion evidence is unrelated or ambiguous.
func TestDeletionProofRejectsOtherGUIDAndMixedAbsence(t *testing.T) {
	guid, _ := shared.ParseGUID(fixtureGUID)
	other, _ := shared.ParseGUID("a3e9767a-fb25-4d2c-849a-14d672c61005")
	tombstone := shared.Object{Attributes: map[string][][]byte{"objectguid": {other.Bytes()}, "isdeleted": {[]byte("TRUE")}}}
	mutation := shared.ObjectMutationResult{
		State: shared.ObjectWriteAccepted, TargetGUID: &other, Verified: true,
		DeletionReadback: &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}},
	}
	if confirmedDeletion(fixtureGUID, mutation, nil) {
		t.Fatal("another object's positive tombstone removed this resource")
	}
	if confirmedDeletion(fixtureGUID, mutation, shared.ErrDirectoryObjectNotFound) {
		t.Fatal("a mismatched target was treated as pure absence")
	}
	mutation.TargetGUID = &guid
	if confirmedDeletion(fixtureGUID, mutation, nil) {
		t.Fatal("tombstone GUID differed from the retained target")
	}
	tombstone.Attributes["objectguid"] = [][]byte{guid.Bytes()}
	mutation.DeletionReadback.Objects[0] = tombstone
	if !confirmedDeletion(fixtureGUID, mutation, nil) {
		t.Fatal("complete same-GUID deletion proof was rejected")
	}
	if !confirmedDeletion(fixtureGUID, shared.ObjectMutationResult{}, shared.ErrDirectoryObjectNotFound) {
		t.Fatal("pure confirmed absence was rejected")
	}
	denied := errors.Join(shared.ErrDirectoryObjectNotFound, &shared.LDAPError{Code: 50})
	if confirmedDeletion(fixtureGUID, shared.ObjectMutationResult{}, denied) {
		t.Fatal("absence mixed with access denial removed state")
	}
}
