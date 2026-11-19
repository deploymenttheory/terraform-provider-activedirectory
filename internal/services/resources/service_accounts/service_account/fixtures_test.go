// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/service_accounts"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	featuremocks "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/service_accounts/service_account/mocks"
)

const fixtureGUID = "dc93656c-f745-4926-9bbb-31e7bbecb68b"

func textElements(values []string) []attr.Value {
	result := make([]attr.Value, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}
	return result
}

// creationModel provides explicit required inputs and typed null collections without inventing optional visibility.
func creationModel() ServiceAccountResourceModel {
	return ServiceAccountResourceModel{
		Name: types.StringValue("initial"), ParentDN: types.StringValue("OU=first,DC=unit,DC=invalid"),
		Kind: types.StringValue("group_managed"), SAMAccountName: types.StringValue("initial$"), DNSHostName: types.StringValue("initial.unit.invalid"),
		PasswordWOVersion: types.Int64Value(0), OutboundAuthenticationOnly: types.BoolValue(false),
		KerberosEncryptionTypes: types.SetNull(types.StringType), ServicePrincipalNames: types.SetNull(types.StringType),
		PasswordReaderSIDs: types.SetNull(types.StringType), ClearProperties: types.SetNull(types.StringType),
	}
}

type unitDirectory struct {
	mu                sync.Mutex
	guid              string
	model             ServiceAccountResourceModel
	exists            bool
	calls             map[string]int
	borrowedPasswords [][]byte
	concurrentDNS     bool
}

// newUnitDirectory supplies explicit callbacks only; it never opens an LDAP connection.
func newUnitDirectory() *unitDirectory {
	return &unitDirectory{guid: fixtureGUID, model: creationModel(), calls: map[string]int{}}
}

// snapshot simulates typed SDK observations, including an absent SPN attribute after an acknowledged clear.
func (d *unitDirectory) snapshot(t *testing.T) *service_accounts.ServiceAccount {
	t.Helper()
	properties, control, diagnostics := constructProperties(&d.model)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	guid, _ := shared.ParseGUID(d.guid)
	name := d.model.Name.ValueString()
	dn := "CN=" + name + "," + d.model.ParentDN.ValueString()
	class, kind := "msDS-GroupManagedServiceAccount", service_accounts.ServiceAccountGroupManaged
	if d.model.Kind.ValueString() == "standalone" {
		class, kind = "msDS-ManagedServiceAccount", service_accounts.ServiceAccountStandalone
	}
	object := shared.Object{
		DistinguishedName:   dn,
		RequestedProperties: []string{"description", "displayName", "dNSHostName", "sAMAccountName", "wWWHomePage"},
		Attributes: map[string][][]byte{
			"objectguid": {guid.Bytes()}, "name": {[]byte(name)}, "objectclass": {[]byte("top"), []byte("computer"), []byte(class)},
		},
	}
	ignored := ServiceAccountResourceModel{}
	for _, field := range textProperties(&ignored, &properties) {
		if *field.target != nil {
			object.Attributes[strings.ToLower(field.ldap)] = [][]byte{[]byte(**field.target)}
		}
	}
	account := &service_accounts.ServiceAccount{
		Kind: kind, GUID: &guid, Name: &name, DistinguishedName: dn, Raw: object,
		SupportedEncryptionTypes: properties.KerberosEncryptionTypes, CompoundIdentitySupported: properties.CompoundIdentitySupported, AccountExpires: properties.AccountExpires,
		AccountControl: accounts.AccountControlState{
			Enabled: control.Enabled, AccountNotDelegated: control.AccountNotDelegated, AllowReversiblePasswordEncryption: control.AllowReversiblePasswordEncryption,
			DoesNotRequirePreAuth: control.DoesNotRequirePreAuth, HomeDirectoryRequired: control.HomeDirectoryRequired, MNSLogonAccount: control.MNSLogonAccount,
			SmartcardLogonRequired: control.SmartcardLogonRequired, TrustedForDelegation: control.TrustedForDelegation,
			TrustedToAuthForDelegation: control.TrustedToAuthForDelegation, UseDESKeyOnly: control.UseDESKeyOnly,
		},
	}
	if properties.ServicePrincipalNames != nil && len(*properties.ServicePrincipalNames) > 0 {
		account.ServicePrincipalNames.State = shared.AttributePresent
		for _, value := range *properties.ServicePrincipalNames {
			account.ServicePrincipalNames.Values = append(account.ServicePrincipalNames.Values, shared.NewRawValue([]byte(value)))
		}
	}
	if kind == service_accounts.ServiceAccountGroupManaged {
		account.PasswordReaderDescriptorReturned = true
		for _, element := range d.model.PasswordReaderSIDs.Elements() {
			sid, err := shared.ParseSID(element.(types.String).ValueString())
			if err != nil {
				t.Fatal(err)
			}
			account.PasswordReaders = append(account.PasswordReaders, sid)
		}
		days := uint32(30)
		if !d.model.ManagedPasswordIntervalDays.IsNull() {
			days = uint32(d.model.ManagedPasswordIntervalDays.ValueInt64())
		}
		account.ManagedPasswordIntervalDays = &days
	}
	return account
}

func (d *unitDirectory) mutation(t *testing.T) shared.ObjectMutationResult {
	account := d.snapshot(t)
	return shared.ObjectMutationResult{
		State: shared.ObjectWriteAccepted, TargetGUID: account.GUID, TargetDN: account.DistinguishedName,
		After: &account.Raw, Verified: true, ReadbackComplete: true,
	}
}

// apply copies selected SDK arguments into synthetic observations; no unrequested property is invented.
func (d *unitDirectory) apply(request service_accounts.UpdateServiceAccountRequest) {
	for _, field := range textProperties(&d.model, &request.Properties) {
		if *field.target != nil {
			*field.value = types.StringValue(**field.target)
		}
	}
	for _, field := range boolProperties(&d.model, &request.Control) {
		if *field.target != nil {
			*field.value = types.BoolValue(**field.target)
		}
	}
	for _, change := range request.Changes {
		for _, field := range textProperties(&d.model, &request.Properties) {
			if change.Kind == shared.AttributeClear && strings.EqualFold(change.Attribute, field.ldap) {
				*field.value = types.StringNull()
			}
		}
	}
	if request.Properties.AccountExpires != nil {
		d.model.AccountExpires = types.Int64Value(int64(*request.Properties.AccountExpires))
	}
	if request.Properties.CompoundIdentitySupported != nil {
		d.model.CompoundIdentitySupported = types.BoolValue(*request.Properties.CompoundIdentitySupported)
	}
	if request.Properties.KerberosEncryptionTypes != nil {
		d.model.KerberosEncryptionTypes = observedEncryptionTypes(*request.Properties.KerberosEncryptionTypes)
	}
	if request.Properties.ServicePrincipalNames != nil {
		d.model.ServicePrincipalNames = types.SetValueMust(types.StringType, textElements(*request.Properties.ServicePrincipalNames))
	}
	if request.PasswordReaders != nil {
		values := make([]string, 0, len(*request.PasswordReaders))
		for _, reader := range *request.PasswordReaders {
			values = append(values, reader.String())
		}
		d.model.PasswordReaderSIDs = types.SetValueMust(types.StringType, textElements(values))
	}
}

// callbacks checks stable GUID selection and supplies the resource's complete synthetic lifecycle.
func (d *unitDirectory) callbacks(t *testing.T) *featuremocks.Service {
	t.Helper()
	check := func(target shared.Identity) {
		if target.Form() != shared.IdentityFormGUID || target.String() != d.guid {
			t.Error("operation did not retain the stable GUID")
		}
	}
	return &featuremocks.Service{
		Read: func(_ context.Context, target shared.Identity, _ shared.ResolveIdentityOptions) (service_accounts.ServiceAccountResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(target)
			if !d.exists {
				return service_accounts.ServiceAccountResult{}, shared.ErrDirectoryObjectNotFound
			}
			account := d.snapshot(t)
			return service_accounts.ServiceAccountResult{Account: account, Observation: shared.IdentityResult{Object: &account.Raw, Complete: true}}, nil
		},
		Create: func(_ context.Context, request service_accounts.CreateServiceAccountRequest) (service_accounts.ServiceAccountCreationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.calls["create"]++
			if d.exists {
				return service_accounts.ServiceAccountCreationResult{}, shared.ErrObjectAlreadyExists
			}
			guid, _ := shared.ParseGUID(fixtureGUID)
			guid[15] += byte(d.calls["create"] - 1)
			d.guid = guid.String()
			d.model = creationModel()
			d.model.Name = types.StringValue(request.Name)
			d.model.ParentDN = types.StringValue(request.Parent.String())
			d.model.Kind = types.StringValue("group_managed")
			if request.Kind == service_accounts.ServiceAccountStandalone {
				d.model.Kind = types.StringValue("standalone")
			}
			d.model.DNSHostName = types.StringNull()
			d.model.OutboundAuthenticationOnly = types.BoolValue(request.OutboundAuthenticationOnly)
			if request.ManagedPasswordIntervalDays != nil {
				d.model.ManagedPasswordIntervalDays = types.Int64Value(int64(*request.ManagedPasswordIntervalDays))
			}
			d.apply(service_accounts.UpdateServiceAccountRequest{Properties: request.Properties, Control: request.Control})
			if request.Control.Enabled == nil {
				d.model.Enabled = types.BoolValue(true)
			}
			if request.Kind == service_accounts.ServiceAccountGroupManaged {
				d.apply(service_accounts.UpdateServiceAccountRequest{PasswordReaders: &request.PasswordReaders})
			}
			d.exists = true
			result := service_accounts.ServiceAccountCreationResult{Creation: d.mutation(t), Account: d.snapshot(t), Complete: true}
			if request.Kind == service_accounts.ServiceAccountStandalone {
				d.calls["password"]++
				d.borrowedPasswords = append(d.borrowedPasswords, request.InitialPassword)
				result.Password = &accounts.PasswordResult{State: shared.ObjectWriteAccepted, TargetGUID: result.Creation.TargetGUID}
				if request.Control.Enabled == nil || *request.Control.Enabled {
					stage := d.mutation(t)
					result.Enable = &stage
				}
			}
			return result, nil
		},
		Update: func(_ context.Context, request service_accounts.UpdateServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["update"]++
			d.apply(request)
			return service_accounts.ServiceAccountMutationResult{Mutation: d.mutation(t), Account: d.snapshot(t)}, nil
		},
		Rename: func(_ context.Context, request shared.RenameObjectRequest) (shared.ObjectMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["rename"]++
			d.model.Name = types.StringValue(request.Name)
			if d.concurrentDNS {
				d.model.DNSHostName = types.StringValue("concurrent.unit.invalid")
			}
			return d.mutation(t), nil
		},
		Move: func(_ context.Context, request shared.MoveObjectRequest) (shared.ObjectMoveResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["move"]++
			d.model.ParentDN = types.StringValue(request.TargetParent.String())
			if request.Destination != nil {
				t.Error("same-domain move selected another destination")
			}
			return shared.ObjectMoveResult{Mutation: d.mutation(t)}, nil
		},
		Delete: func(_ context.Context, request service_accounts.DeleteServiceAccountRequest) (service_accounts.ServiceAccountMutationResult, error) {
			d.mu.Lock()
			defer d.mu.Unlock()
			check(request.Identity)
			d.calls["delete"]++
			result := d.mutation(t)
			d.exists = false
			tombstone := shared.Object{Attributes: map[string][][]byte{"objectguid": {result.TargetGUID.Bytes()}, "isdeleted": {[]byte("TRUE")}}}
			result.DeletionReadback = &shared.SearchResult{Complete: true, Objects: []shared.Object{tombstone}}
			return service_accounts.ServiceAccountMutationResult{Mutation: result}, nil
		},
	}
}
