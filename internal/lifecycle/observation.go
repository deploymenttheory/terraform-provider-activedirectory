// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package lifecycle preserves SDK observations and write outcomes for Framework resources.
package lifecycle

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
)

// ConfirmedAbsent accepts only a pure SDK not-found error with no contradictory returned objects.
// Joined denied, incomplete, transport, cancellation or unknown failures never authorize state removal.
func ConfirmedAbsent(observed shared.IdentityResult, err error) bool {
	return observed.Object == nil && len(observed.Candidates) == 0 && onlyReadErrors(err, false)
}

func onlyReadErrors(err error, allowIncomplete bool) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !onlyReadErrors(child, allowIncomplete) {
				return false
			}
		}
		return true
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return onlyReadErrors(wrapped, allowIncomplete)
	}
	class := shared.Classify(err)
	return class == shared.ClassNotFound || (allowIncomplete && class == shared.ClassIncomplete)
}

// StringObservation retains the SDK retrieval state separately from a usable single text value.
// Missing/hidden and ranged attributes cannot be mistaken for an observed empty string or explicit clear.
type StringObservation struct {
	Value       types.String
	State       shared.AttributeState
	Usable      bool
	Diagnostics diag.Diagnostics
}

// ObserveString converts only a complete returned single value. Callers retain prior state when unusable.
func ObserveString(object shared.Object, attribute string) StringObservation {
	values := object.Values(attribute)
	result := StringObservation{Value: types.StringNull(), State: values.State}
	if values.State != shared.AttributePresent {
		return result
	}
	value, err := values.Single()
	if err == nil {
		var text string
		text, err = value.Text()
		if err == nil {
			result.Value = types.StringValue(text)
			result.Usable = true
			return result
		}
	}
	result.Diagnostics.AddError("Invalid directory observation", "A selected single-valued attribute did not return one valid text value. Previous state is retained.")
	return result
}

// PlannedString preserves omitted versus explicitly empty input and refuses unknown values before SDK I/O.
// Feature request construction decides whether a null update means omission or an explicit AttributeClear.
func PlannedString(value types.String) (*string, diag.Diagnostics) {
	if value.IsUnknown() {
		var diagnostics diag.Diagnostics
		diagnostics.AddError("Unknown directory input", "The planned value must be known before submitting a directory mutation.")
		return nil, diagnostics
	}
	if value.IsNull() {
		return nil, nil
	}
	text := value.ValueString()
	return &text, nil
}
