// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"context"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	mocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/objects/object/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

// unitDirectory supplies explicit public SDK callbacks; it does not simulate LDAP schema validation or security.
type unitDirectory struct {
	mu     sync.Mutex
	object shared.Object
	exists bool
	parent string
	name   string
	class  string
	calls  map[string]int
}

func newUnitDirectory() *unitDirectory { return &unitDirectory{calls: map[string]int{}} }

// snapshot copies raw fixture values so provider observations cannot mutate the synthetic directory.
func (d *unitDirectory) snapshot(properties []string) shared.Object {
	guid, _ := shared.ParseGUID(fixtureGUID)
	rdn := "CN="
	if d.class == "organizationalUnit" {
		rdn = "OU="
	}
	dn := rdn + d.name + "," + d.parent
	object := shared.Object{DistinguishedName: dn, RequestedProperties: slices.Clone(properties), Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "name": {[]byte(d.name)}, "objectclass": {[]byte("top"), []byte(d.class)}, "distinguishedname": {[]byte(dn)},
	}}
	for name, values := range d.object.Attributes {
		for _, value := range values {
			object.Attributes[name] = append(object.Attributes[name], slices.Clone(value))
		}
	}
	return object
}

// mutation returns an acknowledged complete same-GUID SDK observation.
func (d *unitDirectory) mutation(properties []string) shared.ObjectMutationResult {
	guid, _ := shared.ParseGUID(fixtureGUID)
	object := d.snapshot(properties)
	return shared.ObjectMutationResult{
		State: shared.ObjectWriteAccepted, TargetGUID: &guid, TargetDN: object.DistinguishedName,
		After: &object, ReadbackComplete: true, Verified: true,
	}
}

// callbacks rejects unexpected identity broadening and exposes strict, individually counted mutation stages.
func (d *unitDirectory) callbacks(t *testing.T) *mocks.Service {
	t.Helper()
	check := func(target shared.Identity) {
		if target.Form() != shared.IdentityFormGUID || target.String() != fixtureGUID {
			t.Error("generic mutation lost stable GUID identity")
		}
	}
	return &mocks.Service{
		Read: func(_ context.Context, target shared.Identity, options shared.ResolveIdentityOptions) (shared.IdentityResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(target)
			if !d.exists {
				return shared.IdentityResult{}, shared.ErrDirectoryObjectNotFound
			}
			object := d.snapshot(options.Properties)
			return shared.IdentityResult{Complete: true, Object: &object}, nil
		},
		Create: func(_ context.Context, request shared.CreateObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return shared.ObjectMutationResult{}, shared.ErrObjectAlreadyExists
			}
			if request.Parent.Form() != shared.IdentityFormDN || request.Options.ExpectedVersion != nil {
				t.Error("creation did not retain an explicit parent or introduced unrequested controls")
			}
			d.name, d.parent, d.class, d.exists = request.Name, request.Parent.String(), request.Class, true
			d.object.Attributes = map[string][][]byte{"displayname": {[]byte("unowned server value")}}
			for name, values := range request.Attributes {
				for _, value := range values {
					d.object.Attributes[name] = append(d.object.Attributes[name], value.Bytes())
				}
			}
			return d.mutation(request.Options.Properties), nil
		},
		Rename: func(_ context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["rename"]++
			d.name = request.Name
			return d.mutation(request.Options.Properties), nil
		},
		Move: func(_ context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["move"]++
			if request.Destination != nil || request.TargetParent.Form() != shared.IdentityFormDN {
				t.Error("ordinary move introduced cross-domain delegation or inferred its parent")
			}
			d.parent = request.TargetParent.String()
			return shared.ObjectMoveResult{Mutation: d.mutation(request.Options.Properties)}, nil
		},
		Update: func(_ context.Context, request shared.UpdateObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["update"]++
			unverified := false
			for _, change := range request.Changes {
				values := d.object.Attributes[change.Attribute]
				switch change.Kind {
				case shared.AttributeSet:
					values = nil
					for _, value := range change.Values {
						values = append(values, value.Bytes())
					}
				case shared.AttributeAddValues:
					for _, value := range change.Values {
						if slices.ContainsFunc(values, func(existing []byte) bool { return slices.Equal(existing, value.Bytes()) }) {
							t.Error("provider resubmitted a positively observed existing add value")
						}
						values = append(values, value.Bytes())
					}
				case shared.AttributeRemoveValues:
					for _, value := range change.Values {
						values = slices.DeleteFunc(values, func(existing []byte) bool { return slices.Equal(existing, value.Bytes()) })
					}
				case shared.AttributeClear:
					values, unverified = nil, true
				default:
					t.Error("unexpected raw operation")
				}
				if len(values) == 0 {
					delete(d.object.Attributes, change.Attribute)
					unverified = true
				} else {
					d.object.Attributes[change.Attribute] = values
				}
			}
			result := d.mutation(request.Options.Properties)
			if unverified {
				result.Verified = false
				return result, shared.ErrObjectUnverifiable
			}
			return result, nil
		},
		Delete: func(_ context.Context, request shared.DeleteObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["delete"]++
			if request.Recursive {
				t.Error("resource enabled recursive deletion")
			}
			if !d.exists {
				return shared.ObjectMutationResult{}, shared.ErrDirectoryObjectNotFound
			}
			result := d.mutation(nil)
			d.exists = false
			tombstone := shared.Object{Attributes: map[string][][]byte{"objectguid": {result.TargetGUID.Bytes()}, "isdeleted": {[]byte("TRUE")}}}
			result.After, result.AbsentAfter = nil, true
			result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}}
			return result, nil
		},
	}
}

// unitConfiguration reads the feature-local Terraform fixture with synthetic protected-auth credentials.
func unitConfiguration(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile("tests/terraform/unit/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
