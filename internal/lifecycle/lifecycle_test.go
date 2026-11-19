// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

const testID = "a3e9767a-200e-43dd-bb1d-e5ab400e5d6f"

func TestConfirmedAbsenceRejectsDeniedPartialAndContradictoryReads(t *testing.T) {
	notFound := shared.ErrDirectoryObjectNotFound
	if !ConfirmedAbsent(shared.IdentityResult{}, fmt.Errorf("read by GUID: %w", notFound)) {
		t.Fatal("SDK confirmed not-found was not accepted")
	}
	for _, err := range []error{nil, shared.ErrIncompleteResult, shared.ErrObjectWritePermission, context.Canceled,
		errors.Join(notFound, shared.ErrObjectWritePermission), errors.Join(notFound, shared.ErrIncompleteResult),
		errors.Join(notFound, context.DeadlineExceeded), errors.Join(notFound, errors.New("unclassified failure"))} {
		if ConfirmedAbsent(shared.IdentityResult{}, err) {
			t.Errorf("unsafe absence accepted: %v", err)
		}
	}
	for _, result := range []shared.IdentityResult{
		{Object: &shared.Object{}}, {Candidates: []shared.Object{{}}},
	} {
		if ConfirmedAbsent(result, notFound) {
			t.Fatal("returned objects were discarded as absent")
		}
	}
}

func TestAttributeObservationPreservesHiddenPartialAndExplicitEmptyValues(t *testing.T) {
	for _, row := range []struct {
		object shared.Object
		state  shared.AttributeState
		usable bool
		value  string
	}{
		{shared.Object{}, shared.AttributeUnloaded, false, ""},
		{shared.Object{RequestedProperties: []string{"description"}}, shared.AttributeAbsent, false, ""},
		{shared.Object{Attributes: map[string][][]byte{"description;range=0-0": {[]byte("partial")}}}, shared.AttributePartial, false, ""},
		{shared.Object{Attributes: map[string][][]byte{"description": {[]byte("")}}}, shared.AttributePresent, true, ""},
		{shared.Object{Attributes: map[string][][]byte{"description": {[]byte("observed")}}}, shared.AttributePresent, true, "observed"},
	} {
		observed := ObserveString(row.object, "description")
		if observed.State != row.state || observed.Usable != row.usable || observed.Value.ValueString() != row.value || observed.Diagnostics.HasError() {
			t.Fatalf("observation state was lost: %#v", observed)
		}
	}
	if _, diagnostics := PlannedString(types.StringUnknown()); !diagnostics.HasError() {
		t.Fatal("unknown plan accepted before mutation")
	}
	if value, diagnostics := PlannedString(types.StringNull()); value != nil || diagnostics.HasError() {
		t.Fatal("omitted plan was converted into a value")
	}
	if value, diagnostics := PlannedString(types.StringValue("")); value == nil || *value != "" || diagnostics.HasError() {
		t.Fatal("explicit empty plan was lost")
	}
}

func TestMutationRetainsAcceptedAndUncertainIdentityWithoutReplayingWrites(t *testing.T) {
	guid, err := shared.ParseGUID(testID)
	if err != nil {
		t.Fatal(err)
	}
	result := shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, TargetDN: "OU=owned,DC=unit,DC=invalid"}
	failure := errors.Join(shared.ErrObjectReadback, errors.New("private-fixture-password"))
	for _, previous := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(testID)} {
		outcome := ObserveMutation(previous, result, failure)
		if outcome.ID.ValueString() != testID || !outcome.Accepted || !outcome.ReconcileRequired || !outcome.Diagnostics.HasError() {
			t.Fatalf("accepted identity not retained: %#v", outcome)
		}
		for _, diagnostic := range outcome.Diagnostics {
			if strings.Contains(diagnostic.Detail(), "private-fixture-password") {
				t.Fatal("peer diagnostic leaked")
			}
		}
	}
	result.State = shared.ObjectWriteUncertain
	outcome := ObserveMutation(types.StringValue(testID), result, shared.ErrObjectWriteUncertain)
	if outcome.ID.ValueString() != testID || !outcome.ReconcileRequired || outcome.Accepted {
		t.Fatal("uncertain effect was discarded or recorded as accepted")
	}
	result.State = shared.ObjectWriteAccepted
	result.TargetGUID = nil
	outcome = ObserveMutation(types.StringUnknown(), result, failure)
	if !outcome.ID.IsNull() || outcome.RecoveryDN != result.TargetDN || !outcome.Diagnostics.HasError() {
		t.Fatal("GUID-less creation lost its recovery locator")
	}
	result.TargetGUID = &guid
	result.After = &shared.Object{Attributes: map[string][][]byte{"objectguid": {guid.Bytes()}}}
	result.TargetGUID = nil
	outcome = ObserveMutation(types.StringNull(), result, failure)
	if outcome.ID.ValueString() != testID {
		t.Fatal("known raw readback GUID was discarded")
	}
	result.TargetGUID = &guid
	result.After = nil
	otherID := "b3e9767a-200e-43dd-bb1d-e5ab400e5d6f"
	outcome = ObserveMutation(types.StringValue(otherID), result, nil)
	if outcome.ID.ValueString() != otherID || !outcome.Diagnostics.HasError() {
		t.Fatal("mutation changed stable identity")
	}
	outcome = ObserveMutation(types.StringValue(testID), result, shared.ErrObjectUnverifiable)
	if outcome.Diagnostics.HasError() || outcome.ReconcileRequired || len(outcome.Diagnostics) != 1 {
		t.Fatal("acknowledged unverifiable clear was treated as an uncertain write")
	}
	result.Verified = true
	outcome = ObserveMutation(types.StringNull(), result, nil)
	if outcome.Diagnostics.HasError() || outcome.ID.ValueString() != testID || outcome.ReconcileRequired {
		t.Fatal("verified creation not accepted")
	}
	result.State = shared.ObjectWriteRejected
	outcome = ObserveMutation(types.StringNull(), result, shared.ErrObjectAlreadyExists)
	if !outcome.ID.IsNull() || !outcome.Diagnostics.HasError() {
		t.Fatal("rejected create adopted a collision")
	}
}

func TestDeletionRequiresPositiveSameGUIDObservation(t *testing.T) {
	guid, err := shared.ParseGUID(testID)
	if err != nil {
		t.Fatal(err)
	}
	result := shared.ObjectMutationResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid, Verified: true, AbsentAfter: true}
	if ConfirmedDeletion(result, nil) {
		t.Fatal("empty live search was treated as deletion proof")
	}
	result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{{Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "isdeleted": {[]byte("TRUE")},
	}}}}
	if !ConfirmedDeletion(result, nil) {
		t.Fatal("verified same-GUID tombstone rejected")
	}
	result.DeletionReadback.Complete = false
	if ConfirmedDeletion(result, nil) {
		t.Fatal("incomplete deletion observation accepted")
	}
	result.DeletionReadback.Complete = true
	if ConfirmedDeletion(result, shared.ErrObjectReadback) {
		t.Fatal("failed deletion observation accepted")
	}
	result.DeletionReadback.Objects[0].Attributes["isdeleted"] = [][]byte{[]byte("FALSE")}
	if ConfirmedDeletion(result, nil) {
		t.Fatal("live object accepted as a tombstone")
	}
}

func TestBoundedReadbackRetainsPartialObservationsAndStopsOnDenial(t *testing.T) {
	policy := ReadbackPolicy{Attempts: 3, Timeout: time.Second}
	count := 0
	read := func(context.Context) (shared.IdentityResult, error) {
		count++
		switch count {
		case 1:
			return shared.IdentityResult{Candidates: []shared.Object{{DistinguishedName: "OU=partial"}}}, shared.ErrIncompleteResult
		case 2:
			return shared.IdentityResult{}, shared.ErrDirectoryObjectNotFound
		default:
			return shared.IdentityResult{Complete: true, Object: &shared.Object{DistinguishedName: "OU=complete"}}, nil
		}
	}
	observations, err := Readback(t.Context(), policy, read, func(shared.IdentityResult) bool { return true })
	if err != nil || count != 3 || len(observations) != 3 || len(observations[0].Result.Candidates) != 1 {
		t.Fatalf("bounded readback lost partial evidence: %v", err)
	}
	count = 0
	_, err = Readback(t.Context(), policy, func(context.Context) (shared.IdentityResult, error) {
		count++
		return shared.IdentityResult{}, errors.Join(shared.ErrDirectoryObjectNotFound, shared.ErrObjectWritePermission)
	}, func(shared.IdentityResult) bool { return true })
	if !errors.Is(err, ErrReadback) || count != 1 {
		t.Fatal("denied read was retried")
	}
	count = 0
	_, err = Readback(t.Context(), policy, func(context.Context) (shared.IdentityResult, error) {
		count++
		return shared.IdentityResult{}, errors.Join(shared.ErrIncompleteResult, errors.New("unclassified read failure"))
	}, func(shared.IdentityResult) bool { return true })
	if !errors.Is(err, ErrReadback) || count != 1 {
		t.Fatal("joined unknown failure was retried as an incomplete read")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	count = 0
	_, err = Readback(ctx, policy, read, func(shared.IdentityResult) bool { return true })
	if !errors.Is(err, context.Canceled) || count != 0 {
		t.Fatal("canceled readback performed I/O")
	}
	policy.Attempts = 6
	_, err = Readback(t.Context(), policy, read, func(shared.IdentityResult) bool { return true })
	if !errors.Is(err, ErrReadback) || count != 0 {
		t.Fatal("unbounded read policy accepted")
	}
}
