// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/computers"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/computers/computer/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

type unitDirectory struct {
	mu                sync.Mutex
	model             ComputerResourceModel
	exists            bool
	calls             map[string]int
	borrowedPasswords [][]byte
}

func newUnitDirectory(t *testing.T) *unitDirectory {
	t.Helper()
	data, err := os.ReadFile("tests/responses/computer.json")
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
	model.Enabled = types.BoolValue(false)
	model.PasswordNotRequired = types.BoolValue(false)
	model.PasswordNeverExpires = types.BoolValue(false)
	model.SmartcardLogonRequired = types.BoolValue(false)
	model.AccountExpires = types.Int64Value(0)
	model.ChangePasswordAtLogon = types.BoolValue(true)
	return &unitDirectory{model: model, calls: map[string]int{}}
}

func (d *unitDirectory) snapshot(t *testing.T) *computers.Computer {
	t.Helper()
	properties, control, diagnostics := constructProperties(&d.model)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	guid, _ := shared.ParseGUID(fixtureGUID)
	name := d.model.Name.ValueString()
	dn := "CN=" + name + "," + d.model.ParentDN.ValueString()
	object := shared.Object{DistinguishedName: dn, Attributes: map[string][][]byte{
		"objectguid": {guid.Bytes()}, "name": {[]byte(name)}, "objectclass": {[]byte("top"), []byte("person"), []byte("user"), []byte("computer")},
	}}
	ignored := ComputerResourceModel{}
	for _, field := range textProperties(&ignored, &properties) {
		if *field.target != nil {
			object.Attributes[strings.ToLower(field.ldap)] = [][]byte{[]byte(**field.target)}
		}
	}
	if properties.AccountExpires != nil {
		object.Attributes["accountexpires"] = [][]byte{[]byte(properties.AccountExpires.LDAPString())}
	}
	return &computers.Computer{
		ComputerProperties: properties,
		AccountControl:     fixtureControl(control, d.model.ChangePasswordAtLogon),
		GUID:               &guid,
		Name:               &name,
		DistinguishedName:  dn,
		Raw:                object,
	}
}

func (d *unitDirectory) mutation(t *testing.T) shared.ObjectMutationResult {
	computer := d.snapshot(t)
	return shared.ObjectMutationResult{
		State:            shared.ObjectWriteAccepted,
		TargetGUID:       computer.GUID,
		TargetDN:         computer.DistinguishedName,
		After:            &computer.Raw,
		Verified:         true,
		ReadbackComplete: true,
	}
}

func (d *unitDirectory) applyProperties(properties computers.ComputerProperties, control accounts.AccountControlPatch, changes []shared.AttributeChange) {
	for _, field := range textProperties(&d.model, &properties) {
		if *field.target != nil {
			*field.value = types.StringValue(**field.target)
		}
	}
	for _, field := range boolProperties(&d.model, &control) {
		if *field.target != nil {
			*field.value = types.BoolValue(**field.target)
		}
	}
	if properties.AccountExpires != nil {
		d.model.AccountExpires = types.Int64Value(int64(*properties.AccountExpires))
	}
	if properties.KerberosEncryptionTypes != nil {
		d.model.KerberosEncryptionTypes = observedEncryptionTypes(*properties.KerberosEncryptionTypes)
	}
	if properties.CompoundIdentitySupported != nil {
		d.model.CompoundIdentitySupported = types.BoolValue(*properties.CompoundIdentitySupported)
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
			t.Fatal("computer operation changed GUID addressing or introduced assertion controls")
		}
	}
	return &featuremocks.Service{
		List: func(_ context.Context, options shared.SearchOptions) (computers.ComputersResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			if options.Base == "" || options.ResultLimit < 1 || options.PageSize < 1 {
				t.Fatal("discovery omitted explicit bounds")
			}
			if !d.exists {
				return computers.ComputersResult{Observation: shared.SearchResult{Complete: true}}, nil
			}
			return computers.ComputersResult{Computers: []computers.Computer{*d.snapshot(t)}, Observation: shared.SearchResult{Complete: true}}, nil
		},
		Read: func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (computers.ComputerResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(target, shared.ObjectMutationOptions{})
			if !d.exists {
				return computers.ComputerResult{}, shared.ErrDirectoryObjectNotFound
			}
			computer := d.snapshot(t)
			return computers.ComputerResult{Computer: computer, Observation: shared.IdentityResult{Object: &computer.Raw, Complete: true}}, nil
		},
		Create: func(_ context.Context, request computers.CreateComputerRequest) (computers.ComputerCreationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return computers.ComputerCreationResult{}, shared.ErrObjectAlreadyExists
			}
			if request.Parent.Form() != shared.IdentityFormDN || request.Options.ExpectedVersion != nil {
				t.Fatal("creation guessed its container or imposed an assertion")
			}
			d.model.Name = types.StringValue(request.Name)
			d.model.ParentDN = types.StringValue(request.Parent.String())
			d.applyProperties(request.Properties, request.Control, nil)
			d.exists = true
			result := computers.ComputerCreationResult{Creation: d.mutation(t), Computer: d.snapshot(t), Complete: true}
			if request.InitialPassword != nil {
				d.calls["password"]++
				d.borrowedPasswords = append(d.borrowedPasswords, request.InitialPassword)
				d.model.ChangePasswordAtLogon = types.BoolValue(false)
				result.Computer = d.snapshot(t)
				result.Password = &accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: result.Creation.TargetGUID}
			}
			if request.Control.Enabled != nil && *request.Control.Enabled {
				enabled := d.mutation(t)
				result.Enable = &enabled
			}
			return result, nil
		},
		Update: func(_ context.Context, request computers.UpdateComputerRequest) (computers.ComputerMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			d.calls["update"]++
			d.applyProperties(request.Properties, request.Control, request.Changes)
			return computers.ComputerMutationResult{Mutation: d.mutation(t), Computer: d.snapshot(t)}, nil
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
				t.Fatal("computer move requested another server")
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
			d.model.ChangePasswordAtLogon = types.BoolValue(false)
			guid, _ := shared.ParseGUID(fixtureGUID)
			return accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: &guid}, nil
		},
		Delete: func(_ context.Context, request computers.DeleteComputerRequest) (computers.ComputerMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity, request.Options)
			if request.Recursive {
				t.Fatal("computer deletion requested subtree removal")
			}
			d.calls["delete"]++
			result := d.mutation(t)
			d.exists = false
			tombstone := shared.Object{Attributes: map[string][][]byte{"objectguid": {result.TargetGUID.Bytes()}, "isdeleted": {[]byte("TRUE")}}}
			result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}}
			return computers.ComputerMutationResult{Mutation: result}, nil
		},
	}
}
