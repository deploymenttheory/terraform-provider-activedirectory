// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

var className = regexp.MustCompile(`\A[a-zA-Z][a-zA-Z0-9-]*\z`)

// constructPlan validates all locally enforceable phases and returns immutable request values before any mutation.
func constructPlan(model *ObjectResourceModel, creating bool) ([]shared.AttributeChange, []string, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	for _, value := range []types.String{model.Name, model.ParentDN, model.ObjectClass} {
		if value.IsNull() || value.IsUnknown() || strings.TrimSpace(value.ValueString()) == "" || !utf8.ValidString(value.ValueString()) ||
			strings.ContainsRune(value.ValueString(), 0) {
			diagnostics.AddError("Invalid object configuration", "name, parent_dn and object_class must be known nonempty directory text before any mutation.")
		}
	}
	if !className.MatchString(model.ObjectClass.ValueString()) {
		diagnostics.AddError("Invalid object class", "object_class must be an explicit schema LDAP display name.")
	}
	if _, err := shared.IdentityByDN(model.ParentDN.ValueString()); err != nil {
		diagnostics.AddError("Invalid object parent", "parent_dn must be an explicit distinguished name; no container is inferred.")
	}
	intent := common.AttributeIntent{}
	for _, field := range []struct {
		value  types.Map
		target *map[string][]string
	}{{model.Attributes, &intent.Set}, {model.AddValues, &intent.Add}, {model.RemoveValues, &intent.Remove}} {
		values, conversion := intentMap(field.value)
		diagnostics.Append(conversion...)
		*field.target = values
	}
	if model.ClearAttributes.IsUnknown() {
		diagnostics.AddError("Unknown object clear intent", "Resolve clear_attributes before any mutation.")
	}
	for _, value := range model.ClearAttributes.Elements() {
		text, ok := value.(types.String)
		if !ok || text.IsUnknown() || text.IsNull() {
			diagnostics.AddError("Unknown object clear attribute", "Each clear attribute must be a known LDAP display name.")
		} else {
			intent.Clear = append(intent.Clear, text.ValueString())
		}
	}
	changes, err := common.ConstructChanges(intent)
	if err != nil {
		diagnostics.AddError(
			"Invalid raw attribute intent",
			"Use canonical base64 sets and nonconflicting LDAP display names. Only disjoint remove/add values may share an attribute.",
		)
	}
	if creating && (len(intent.Remove) != 0 || len(intent.Clear) != 0) {
		diagnostics.AddError("Object removal requires an existing object", "remove_values and clear_attributes apply to an existing or imported object.")
	}
	selected, selection := common.ConstructProperties(model.Properties)
	diagnostics.Append(selection...)
	for _, change := range changes {
		selected = append(selected, change.Attribute)
	}
	slices.Sort(selected)
	selected = slices.Compact(selected)
	if diagnostics.HasError() {
		return nil, nil, diagnostics
	}
	return changes, selected, diagnostics
}

// intentMap rejects unknown or empty configured collections without leaking raw attribute values into diagnostics.
func intentMap(value types.Map) (map[string][]string, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	values := map[string][]string{}
	if value.IsUnknown() {
		diagnostics.AddError("Unknown raw object intent", "Resolve configured raw attribute values before any mutation.")
		return nil, diagnostics
	}
	for name, element := range value.Elements() {
		set, ok := element.(types.Set)
		if !ok || set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
			diagnostics.AddError("Invalid raw object values", "Each configured attribute requires a known nonempty set; one empty base64 string means one zero-length value.")
			continue
		}
		for _, item := range set.Elements() {
			text, ok := item.(types.String)
			if !ok || text.IsNull() || text.IsUnknown() {
				diagnostics.AddError("Unknown raw object value", "Every configured raw value must be known before any mutation.")
			} else {
				values[name] = append(values[name], text.ValueString())
			}
		}
	}
	return values, diagnostics
}

// selectedProperties preserves selection on refresh without validating unrelated required create fields during GUID import.
func selectedProperties(model *ObjectResourceModel) ([]string, diag.Diagnostics) {
	properties := model.Properties
	selected, diagnostics := common.ConstructProperties(properties)
	for _, values := range []types.Map{model.Attributes, model.AddValues, model.RemoveValues} {
		for name := range values.Elements() {
			selected = append(selected, strings.ToLower(name))
		}
	}
	for _, value := range model.ClearAttributes.Elements() {
		if text, ok := value.(types.String); ok && !text.IsUnknown() && !text.IsNull() {
			selected = append(selected, strings.ToLower(text.ValueString()))
		}
	}
	slices.Sort(selected)
	return slices.Compact(selected), diagnostics
}

// nullIntentMap supplies the shared element type for omitted raw ownership maps.
func nullIntentMap() types.Map { return types.MapNull(common.RawSetType()) }

// rawSet converts owned fixture/import values without changing their binary representation.
func rawSet(values []shared.RawValue) types.Set {
	items := make([]attr.Value, 0, len(values))
	for _, value := range common.EncodeValues(values) {
		items = append(items, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, items)
}
