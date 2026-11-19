// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"bytes"
	"errors"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestConstructChangesPreservesExplicitOperations covers set/add/remove/clear and disjoint changes to one attribute.
func TestConstructChangesPreservesExplicitOperations(t *testing.T) {
	t.Parallel()
	changes, err := ConstructChanges(AttributeIntent{
		Set:    map[string][]string{"Description": {""}, "jpegPhoto": {"/wCA"}},
		Remove: map[string][]string{"otherTelephone": {"b2xk"}},
		Add:    map[string][]string{"OTHERtelephone": {"bmV3"}},
		Clear:  []string{"displayName"},
	})
	if err != nil || len(changes) != 5 {
		t.Fatal("valid explicit binary and value-level attribute intent was rejected")
	}
	if changes[0].Attribute != "description" || changes[0].Kind != shared.AttributeSet ||
		len(changes[0].Values) != 1 || len(changes[0].Values[0].Bytes()) != 0 {
		t.Fatal("one zero-length value was converted to a clear")
	}
	if changes[1].Attribute != "displayname" || changes[1].Kind != shared.AttributeClear || len(changes[1].Values) != 0 {
		t.Fatal("explicit clear did not retain its own operation")
	}
	if changes[2].Attribute != "jpegphoto" || !bytes.Equal(changes[2].Values[0].Bytes(), []byte{0xff, 0, 0x80}) {
		t.Fatal("binary attribute bytes were converted to text")
	}
	if changes[3].Kind != shared.AttributeRemoveValues || changes[4].Kind != shared.AttributeAddValues ||
		changes[3].Attribute != "othertelephone" || changes[4].Attribute != "othertelephone" {
		t.Fatal("disjoint value changes were not ordered remove before add")
	}
}

// TestConstructChangesRejectsConflicts requires complete preflight rather than returning a partially valid write.
func TestConstructChangesRejectsConflicts(t *testing.T) {
	t.Parallel()
	for name, intent := range map[string]AttributeIntent{
		"case duplicate":  {Set: map[string][]string{"description": {"YQ=="}, "Description": {"Yg=="}}},
		"set and clear":   {Set: map[string][]string{"description": {"YQ=="}}, Clear: []string{"DESCRIPTION"}},
		"set and add":     {Set: map[string][]string{"description": {"YQ=="}}, Add: map[string][]string{"description": {"Yg=="}}},
		"duplicate clear": {Clear: []string{"description", "Description"}},
		"overlap": {
			Remove: map[string][]string{"description": {"YQ=="}}, Add: map[string][]string{"Description": {"YQ=="}},
		},
		"option":      {Clear: []string{"description;binary"}},
		"numeric oid": {Clear: []string{"1.2.3"}},
		"wildcard":    {Clear: []string{"*"}},
		"guid":        {Clear: []string{"objectGUID"}},
		"dn":          {Clear: []string{"distinguishedName"}},
		"class":       {Clear: []string{"objectClass"}},
		"name":        {Clear: []string{"name"}},
	} {
		t.Run(name, func(t *testing.T) {
			if changes, err := ConstructChanges(intent); changes != nil || !errors.Is(err, ErrAttributeIntent) {
				t.Fatal("conflicting or reserved raw attribute intent produced a partial request")
			}
		})
	}
	changes, err := ConstructChanges(AttributeIntent{
		Set: map[string][]string{"description": {"YQ=="}, "jpegPhoto": {"invalid encoding"}},
	})
	if changes != nil || !errors.Is(err, ErrRawEncoding) {
		t.Fatal("invalid binary input retained a partially constructed request")
	}
}
