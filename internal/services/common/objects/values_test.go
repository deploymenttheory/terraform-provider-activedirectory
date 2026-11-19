// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// TestRawValuesRoundTrip covers non-UTF8 bytes, embedded NUL, zero-length values and independent ownership.
func TestRawValuesRoundTrip(t *testing.T) {
	t.Parallel()
	binary := []byte{0xff, 0, 0x80, 0x01}
	values := []shared.RawValue{shared.NewRawValue(binary), shared.NewRawValue(nil), shared.NewRawValue([]byte("text"))}
	encoded := EncodeValues(values)
	decoded, err := DecodeValues(encoded)
	if err != nil || !slices.Equal(encoded, EncodeValues(decoded)) {
		t.Fatal("raw binary attribute values did not survive a Terraform representation round trip")
	}
	foundBinary, foundEmpty := false, false
	for _, value := range decoded {
		foundBinary = foundBinary || bytes.Equal(value.Bytes(), binary)
		foundEmpty = foundEmpty || len(value.Bytes()) == 0
	}
	if !foundBinary || !foundEmpty {
		t.Fatal("binary or zero-length value was lost or mistaken for an attribute clear")
	}
	encoded[0] = "changed"
	if !slices.Equal(EncodeValues(values), EncodeValues(decoded)) {
		t.Fatal("encoded configuration mutated the SDK value buffers")
	}
}

// TestDecodeRejectsAmbiguousRawValues rejects lossy spellings and conflicting set intent before any write.
func TestDecodeRejectsAmbiguousRawValues(t *testing.T) {
	t.Parallel()
	for _, encoded := range [][]string{nil, {}, {"YQ==", "YQ=="}} {
		if values, err := DecodeValues(encoded); values != nil || !errors.Is(err, ErrRawValues) {
			t.Fatal("empty or duplicate raw set was accepted")
		}
	}
	for _, value := range []string{"YQ", "YR==", "YQ==\n", "not base64", "_w=="} {
		if values, err := DecodeValues([]string{value}); values != nil || !errors.Is(err, ErrRawEncoding) {
			t.Fatal("noncanonical binary value was accepted or leaked into a partial request")
		}
	}
}

// TestEncodeSetBoundary retains the SDK's duplicate observations while making Terraform set order stable.
func TestEncodeSetBoundary(t *testing.T) {
	t.Parallel()
	values := []shared.RawValue{shared.NewRawValue([]byte("z")), shared.NewRawValue(nil), shared.NewRawValue([]byte("z"))}
	encoded := EncodeValues(values)
	if len(encoded) != 2 || encoded[0] != "" || len(values) != 3 || len(values[0].Bytes()) != 1 {
		t.Fatal("Terraform set encoding changed the raw observation or omitted the empty value")
	}
	if _, err := DecodeValues([]string{""}); err != nil {
		t.Fatal("one zero-length value must remain distinct from an empty attribute collection")
	}
}
