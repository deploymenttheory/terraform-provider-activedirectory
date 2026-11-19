// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/lifecycle"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

type ownedAuxiliary struct {
	DN      string
	Name    string
	Class   string
	Removed bool
}

// ownedObject independently checks a GUID, permanent description marker and bounded name/location before fixture effects.
func (l *objectLab) ownedObject(ctx context.Context, guid string) (*shared.Object, error) {
	if l.observer == nil {
		return nil, errLabOwnership
	}
	target, err := shared.IdentityByGUID(guid)
	if err != nil {
		return nil, errLabOwnership
	}
	result, err := l.observer.Directory.ResolveIdentity(ctx, target, shared.ResolveIdentityOptions{
		Properties: []string{"objectGUID", "name", "objectClass", "description", "distinguishedName", "manager", "otherTelephone"},
	})
	if err != nil || !result.Complete || result.Object == nil {
		return nil, errLabOwnership
	}
	returned, valid := common.GUID(*result.Object)
	marker, markerErr := result.Object.Values("description").Single()
	name, nameErr := result.Object.Values("name").Single()
	if !valid || returned != guid || markerErr != nil || string(marker.Bytes()) != l.marker || nameErr != nil ||
		!strings.HasPrefix(string(name.Bytes()), l.prefix+"-") || !strings.HasSuffix(strings.ToLower(result.Object.DistinguishedName), ","+strings.ToLower(l.parentDN)) {
		return nil, errLabOwnership
	}
	classes := result.Object.Values("objectClass")
	if classes.State != shared.AttributePresent || len(classes.Values) == 0 {
		return nil, errLabOwnership
	}
	class := string(classes.Values[len(classes.Values)-1].Bytes())
	if class != "contact" && class != "organizationalUnit" {
		return nil, errLabOwnership
	}
	return result.Object, nil
}

// createAuxiliary creates only a contact or OU below a verified owned parent and registers leaf-only retirement.
func (l *objectLab) createAuxiliary(t *testing.T, class, suffix, parentGUID string) string {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.observer == nil || class != "contact" && class != "organizationalUnit" {
		t.Fatal("independent owned fixture observer or class is unavailable")
	}
	parent, err := shared.IdentityByGUID(parentGUID)
	if err != nil {
		t.Fatal("owned auxiliary parent GUID is invalid")
	}
	if parentGUID == l.parentGUID {
		root, readErr := l.observer.Directory.ResolveIdentity(t.Context(), parent, shared.ResolveIdentityOptions{Properties: []string{"description", "objectGUID"}})
		if readErr != nil || !root.Complete || root.Object == nil {
			t.Fatal("external root did not independently verify")
		}
		marker, markerErr := root.Object.Values("description").Single()
		if markerErr != nil || string(marker.Bytes()) != "sdk-portable-parent:"+l.target.RunID || !strings.EqualFold(root.Object.DistinguishedName, l.parentDN) {
			t.Fatal("external root ownership changed")
		}
	} else if _, readErr := l.ownedObject(t.Context(), parentGUID); readErr != nil {
		t.Fatal("auxiliary parent ownership did not verify")
	}
	name := l.prefix + "-" + suffix
	result, err := l.observer.Directory.CreateObject(t.Context(), shared.CreateObjectRequest{
		Parent: parent, Class: class, Name: name,
		Attributes: map[string][]shared.RawValue{"description": {shared.NewRawValue([]byte(l.marker))}},
	})
	guid := ""
	if result.TargetGUID != nil && *result.TargetGUID != (shared.GUID{}) {
		guid = result.TargetGUID.String()
		l.auxiliaries[guid] = ownedAuxiliary{DN: result.TargetDN, Name: name, Class: class}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := l.removeAuxiliary(ctx, guid); err != nil {
				t.Error("owned auxiliary retirement did not independently verify")
			}
		})
	}
	if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified || guid == "" {
		t.Fatal("owned auxiliary creation did not verify; preserve partial outcome")
	}
	if _, err := l.ownedObject(t.Context(), guid); err != nil {
		t.Fatal("owned auxiliary creation failed independent observation")
	}
	return guid
}

// removeAuxiliary verifies the original GUID/name/class/location and refuses descendants before one explicit deletion.
func (l *objectLab) removeAuxiliary(ctx context.Context, guid string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	fixture, found := l.auxiliaries[guid]
	if !found {
		return errLabOwnership
	}
	if fixture.Removed {
		return nil
	}
	object, err := l.ownedObject(ctx, guid)
	if err != nil || object == nil || !strings.EqualFold(object.DistinguishedName, fixture.DN) {
		return errLabOwnership
	}
	name, nameErr := object.Values("name").Single()
	classes := object.Values("objectClass")
	if nameErr != nil || string(name.Bytes()) != fixture.Name || classes.State != shared.AttributePresent || len(classes.Values) == 0 ||
		string(classes.Values[len(classes.Values)-1].Bytes()) != fixture.Class {
		return errLabOwnership
	}
	children, err := l.observer.Directory.Search(ctx, shared.SearchOptions{Base: object.DistinguishedName, Scope: shared.ScopeOneLevel, PageSize: 2, ResultLimit: 2})
	if err != nil || !children.Complete || children.LimitReached || len(children.Objects) != 0 {
		return errLabOwnership
	}
	target, _ := shared.IdentityByGUID(guid)
	result, err := l.observer.Directory.DeleteObject(ctx, shared.DeleteObjectRequest{Identity: target, Recursive: false})
	if !lifecycle.ConfirmedDeletion(result, err) {
		return errLabOwnership
	}
	fixture.Removed = true
	l.auxiliaries[guid] = fixture
	return nil
}

// mutateOwnedValues supplies out-of-band drift only after independent marker/GUID validation, with no automatic retry.
func (l *objectLab) mutateOwnedValues(t *testing.T, changes []shared.AttributeChange) {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.ownedObject(t.Context(), l.childGUID); err != nil {
		t.Fatal("subject ownership did not verify before independent fixture drift")
	}
	target, _ := shared.IdentityByGUID(l.childGUID)
	result, err := l.observer.Directory.UpdateObject(t.Context(), shared.UpdateObjectRequest{Identity: target, Changes: changes})
	if err != nil || result.State != shared.ObjectWriteAccepted || !result.Verified || result.TargetGUID == nil || result.TargetGUID.String() != l.childGUID {
		t.Fatal("independent fixture drift did not verify; no replay attempted")
	}
}
