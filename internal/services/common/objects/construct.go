// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var (
	ErrAttributeIntent = errors.New("raw attribute intent contains an invalid name or conflicting operations")
	attributeName      = regexp.MustCompile(`\A[a-zA-Z][a-zA-Z0-9-]*\z`)
)

// AttributeIntent distinguishes complete attribute ownership from selected value additions, removals and explicit clears.
// Each value is canonical padded base64, including an empty string for one zero-length directory value.
type AttributeIntent struct {
	Set    map[string][]string
	Add    map[string][]string
	Remove map[string][]string
	Clear  []string
}

// ConstructChanges validates the complete intent before returning a request and never submits a directory write.
// LDAP attribute names are case insensitive. Only a disjoint remove/add pair may target the same attribute.
func ConstructChanges(intent AttributeIntent) ([]shared.AttributeChange, error) {
	changes := make([]shared.AttributeChange, 0, len(intent.Set)+len(intent.Add)+len(intent.Remove)+len(intent.Clear))
	for _, group := range []struct {
		kind   shared.AttributeChangeKind
		values map[string][]string
	}{
		{shared.AttributeSet, intent.Set},
		{shared.AttributeRemoveValues, intent.Remove},
		{shared.AttributeAddValues, intent.Add},
	} {
		for name, values := range group.values {
			if !writableAttribute(name) {
				return nil, ErrAttributeIntent
			}
			raw, err := DecodeValues(values)
			if err != nil {
				return nil, err
			}
			changes = append(changes, shared.AttributeChange{Attribute: strings.ToLower(name), Kind: group.kind, Values: raw})
		}
	}
	for _, name := range intent.Clear {
		if !writableAttribute(name) {
			return nil, ErrAttributeIntent
		}
		changes = append(changes, shared.AttributeChange{Attribute: strings.ToLower(name), Kind: shared.AttributeClear})
	}
	slices.SortFunc(changes, func(a, b shared.AttributeChange) int {
		if order := strings.Compare(a.Attribute, b.Attribute); order != 0 {
			return order
		}
		// The SDK submits remove before add; sorting here also makes the provider request deterministic.
		if a.Kind == shared.AttributeRemoveValues && b.Kind == shared.AttributeAddValues {
			return -1
		}
		if a.Kind == shared.AttributeAddValues && b.Kind == shared.AttributeRemoveValues {
			return 1
		}
		return int(a.Kind) - int(b.Kind)
	})
	if err := validateChanges(changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// writableAttribute reserves object identity, class and naming for their dedicated lifecycle fields.
func writableAttribute(name string) bool {
	if !attributeName.MatchString(name) {
		return false
	}
	switch strings.ToLower(name) {
	case "objectguid", "distinguishedname", "objectclass", "name":
		return false
	default:
		return true
	}
}

// validateChanges rejects overlapping ownership and repeated operations without exposing any raw values in errors.
func validateChanges(changes []shared.AttributeChange) error {
	for i := 1; i < len(changes); i++ {
		previous, current := changes[i-1], changes[i]
		if previous.Attribute != current.Attribute {
			continue
		}
		if previous.Kind != shared.AttributeRemoveValues || current.Kind != shared.AttributeAddValues {
			return ErrAttributeIntent
		}
		removed := make(map[string]bool, len(previous.Values))
		for _, value := range previous.Values {
			removed[string(value.Bytes())] = true
		}
		for _, value := range current.Values {
			if removed[string(value.Bytes())] {
				return ErrAttributeIntent
			}
		}
	}
	return nil
}
