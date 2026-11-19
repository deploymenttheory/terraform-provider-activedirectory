// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/users/user/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

type unitDirectory struct {
	mu                sync.Mutex
	model             UserResourceModel
	exists            bool
	calls             map[string]int
	borrowedPasswords [][]byte
}

func newUnitDirectory(t *testing.T) *unitDirectory {
	t.Helper()
	data, err := os.ReadFile("tests/responses/user.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		GUID   string `json:"guid"`
		Name   string `json:"name"`
		SAM    string `json:"samAccountName"`
		Parent string `json:"parentDn"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	model := creationModel()
	model.ID = types.StringValue(fixture.GUID)
	model.Name = types.StringValue(fixture.Name)
	model.SAMAccountName = types.StringValue(fixture.SAM)
	model.ParentDN = types.StringValue(fixture.Parent)
	model.ObjectClass = types.StringValue("user")
	model.Enabled = types.BoolValue(false)
	model.PasswordNotRequired = types.BoolValue(false)
	model.PasswordNeverExpires = types.BoolValue(false)
	model.SmartcardLogonRequired = types.BoolValue(false)
	model.AccountExpires = types.Int64Value(0)
	return &unitDirectory{model: model, calls: map[string]int{}}
}

func (d *unitDirectory) snapshot(t *testing.T) *users.User {
	t.Helper()
	properties, diagnostics := constructProperties(&d.model)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	guid, _ := shared.ParseGUID(fixtureGUID)
	name := d.model.Name.ValueString()
	dn := "CN=" + name + "," + d.model.ParentDN.ValueString()
	object := shared.Object{DistinguishedName: dn, Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "name": {[]byte(name)}, "objectclass": {[]byte("top"), []byte("person"), []byte("user")},
	}}
	ignored := UserResourceModel{}
	for _, field := range textProperties(&ignored, &properties) {
		if *field.target != nil {
			object.Attributes[strings.ToLower(field.ldap)] = [][]byte{[]byte(**field.target)}
		}
	}
	if properties.AccountExpires != nil {
		object.Attributes["accountexpires"] = [][]byte{[]byte(properties.AccountExpires.LDAPString())}
	}
	return &users.User{UserProperties: properties, GUID: &guid, Name: &name, DistinguishedName: dn, Raw: object}
}

func (d *unitDirectory) mutation(t *testing.T) shared.ObjectMutationResult {
	user := d.snapshot(t)
	return shared.ObjectMutationResult{
		State:            shared.ObjectWriteAccepted,
		TargetGUID:       user.GUID,
		TargetDN:         user.DistinguishedName,
		After:            &user.Raw,
		Verified:         true,
		ReadbackComplete: true,
	}
}

func (d *unitDirectory) applyProperties(properties users.UserProperties, changes []shared.AttributeChange) {
	for _, field := range textProperties(&d.model, &properties) {
		if *field.target != nil {
			*field.value = types.StringValue(**field.target)
		}
	}
	for _, field := range boolProperties(&d.model, &properties) {
		if *field.target != nil {
			*field.value = types.BoolValue(**field.target)
		}
	}
	if properties.AccountExpires != nil {
		d.model.AccountExpires = types.Int64Value(int64(*properties.AccountExpires))
	}
	for _, change := range changes {
		for _, field := range textProperties(&d.model, &properties) {
			if change.Kind == shared.AttributeClear && change.Attribute == field.ldap {
				*field.value = types.StringNull()
			}
		}
	}
}

func (d *unitDirectory) callbacks(t *testing.T) *featuremocks.Service {
	t.Helper()
	check := func(target shared.Identity, options shared.ObjectMutationOptions) {
		if target.Form() != shared.IdentityFormGUID || target.String() != fixtureGUID || options.ExpectedVersion != nil {
			t.Fatal("user operation changed GUID addressing or introduced assertion controls")
		}
	}
	return &featuremocks.Service{
		List: func(_ context.Context, options shared.SearchOptions) (users.UsersResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			if options.Base == "" || options.ResultLimit < 1 || options.PageSize < 1 {
				t.Fatal("discovery omitted explicit bounds")
			}
			if !d.exists {
				return users.UsersResult{Observation: shared.SearchResult{Complete: true}}, nil
			}
			return users.UsersResult{Users: []users.User{*d.snapshot(t)}, Observation: shared.SearchResult{Complete: true}}, nil
		},
		Read: func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (users.UserResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(target, shared.ObjectMutationOptions{})
			if !d.exists {
				return users.UserResult{}, shared.ErrDirectoryObjectNotFound
			}
			user := d.snapshot(t)
			return users.UserResult{User: user, Observation: shared.IdentityResult{Object: &user.Raw, Complete: true}}, nil
		},
		Create: func(_ context.Context, request users.CreateUserRequest) (users.UserCreationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return users.UserCreationResult{}, shared.ErrObjectAlreadyExists
			}
			if request.Parent.Form() != shared.IdentityFormDN || request.Options.ExpectedVersion != nil {
				t.Fatal("creation guessed its container or imposed an assertion")
			}
			d.model.Name = types.StringValue(request.Name)
			d.model.ParentDN = types.StringValue(request.Parent.String())
			d.applyProperties(request.Properties, nil)
			d.exists = true
			result := users.UserCreationResult{Creation: d.mutation(t), User: d.snapshot(t), Complete: true}
			if request.InitialPassword != nil {
				d.calls["password"]++
				d.borrowedPasswords = append(d.borrowedPasswords, request.InitialPassword)
				result.Password = &accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: result.Creation.TargetGUID}
			}
			if request.Properties.Enabled != nil && *request.Properties.Enabled {
				enabled := d.mutation(t)
				result.Enable = &enabled
			}
			return result, nil
		},
		Update: func(_ context.Context, request users.UpdateUserRequest) (users.UserMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			d.calls["update"]++
			d.applyProperties(request.Properties, request.Changes)
			return users.UserMutationResult{Mutation: d.mutation(t), User: d.snapshot(t)}, nil
		},
		Rename: func(_ context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			d.calls["rename"]++
			d.model.Name = types.StringValue(request.Name)
			return d.mutation(t), nil
		},
		Move: func(_ context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			if request.Destination != nil {
				t.Fatal("user move requested another server")
			}
			d.calls["move"]++
			d.model.ParentDN = types.StringValue(request.TargetParent.String())
			return shared.ObjectMoveResult{Mutation: d.mutation(t)}, nil
		},
		Password: func(_ context.Context, request accounts.PasswordResetRequest) (accounts.PasswordResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, shared.ObjectMutationOptions{})
			d.calls["password"]++
			d.borrowedPasswords = append(d.borrowedPasswords, request.NewPassword)
			guid, _ := shared.ParseGUID(fixtureGUID)
			return accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid}, nil
		},
		Delete: func(_ context.Context, request users.DeleteUserRequest) (users.UserMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			d.calls["delete"]++
			result := d.mutation(t)
			d.exists = false
			tombstone := shared.Object{Attributes: map[string][][]byte{"objectguid": {result.TargetGUID.Bytes()}, "isdeleted": {[]byte("TRUE")}}}
			result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}}
			return users.UserMutationResult{Mutation: result}, nil
		},
	}
}
