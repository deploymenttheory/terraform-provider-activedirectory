// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package objects preserves explicitly selected generic directory values without inferring their LDAP syntax.
package objects

import (
	"encoding/base64"
	"errors"
	"slices"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

var (
	ErrRawEncoding = errors.New("raw attribute values require canonical padded base64")
	ErrRawValues   = errors.New("raw attribute set contains no values or repeats a value")
)

// DecodeValues preserves binary and zero-length values; an empty collection requires an explicit clear operation.
// Values are sorted for deterministic SDK requests. Errors contain no attribute contents.
func DecodeValues(encoded []string) ([]shared.RawValue, error) {
	if len(encoded) == 0 {
		return nil, ErrRawValues
	}
	ordered := slices.Clone(encoded)
	slices.Sort(ordered)
	values := make([]shared.RawValue, 0, len(ordered))
	for i, value := range ordered {
		if i > 0 && value == ordered[i-1] {
			return nil, ErrRawValues
		}
		decoded, err := base64.StdEncoding.Strict().DecodeString(value)
		if err != nil || base64.StdEncoding.EncodeToString(decoded) != value {
			return nil, ErrRawEncoding
		}
		values = append(values, shared.NewRawValue(decoded))
	}
	return values, nil
}

// EncodeValues returns a deterministic Terraform set representation without converting directory bytes to text.
// Duplicate directory values collapse only at this explicit set boundary; the SDK observation remains untouched.
func EncodeValues(values []shared.RawValue) []string {
	encoded := make([]string, 0, len(values))
	for _, value := range values {
		encoded = append(encoded, base64.StdEncoding.EncodeToString(value.Bytes()))
	}
	slices.Sort(encoded)
	return slices.Compact(encoded)
}
