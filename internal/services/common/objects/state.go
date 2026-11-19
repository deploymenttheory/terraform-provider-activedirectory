// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/identity"
)

// RawSetType represents uninterpreted directory bytes as a set of canonical base64 strings.
func RawSetType() types.SetType { return types.SetType{ElemType: types.StringType} }

// AttributeTypes describes observations shared by generic resource, data-source and discovery snapshots.
func AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "parent_dn": types.StringType,
		"distinguished_name": types.StringType, "object_class": types.StringType,
		"object_classes":      types.ListType{ElemType: types.StringType},
		"observed_attributes": types.MapType{ElemType: RawSetType()},
		"attribute_status":    types.MapType{ElemType: types.StringType},
		"attribute_ranges":    types.MapType{ElemType: RawSetType()},
	}
}

// GUID requires a usable selected objectGUID; distinguished names and raw candidates never substitute for identity.
func GUID(object shared.Object) (string, bool) {
	value, err := object.Values("objectGUID").Single()
	if err != nil {
		return "", false
	}
	guid, err := value.GUID()
	if err != nil || guid == (shared.GUID{}) {
		return "", false
	}
	return guid.String(), true
}

// Project retains raw binary values and visibility states, keeping ranged values separate from complete attributes.
func Project(object shared.Object, properties []string) (map[string]attr.Value, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	id, valid := GUID(object)
	name := singleText(object, "name")
	classes, classDiagnostics := objectClasses(object)
	diagnostics.Append(classDiagnostics...)
	dn := object.DistinguishedName
	if observed := singleText(object, "distinguishedName"); !observed.IsNull() {
		dn = observed.ValueString()
	}
	parent, parentValid := identity.ParentDN(dn)
	if !valid || name.IsNull() || !parentValid {
		diagnostics.AddError("Incomplete object identity", "A nonzero GUID, name and valid directory-observed location are required. No import identity is emitted.")
		return nil, diagnostics
	}
	class := types.StringNull()
	if !classes.IsNull() {
		class = classes.Elements()[len(classes.Elements())-1].(types.String)
	}
	attributes, status, ranges := rawAttributes(object, properties)
	values := map[string]attr.Value{
		"id": types.StringValue(id), "name": name, "parent_dn": types.StringValue(parent),
		"distinguished_name": types.StringValue(dn), "object_class": class, "object_classes": classes,
		"observed_attributes": types.MapValueMust(RawSetType(), attributes),
		"attribute_status":    types.MapValueMust(types.StringType, status),
		"attribute_ranges":    types.MapValueMust(RawSetType(), ranges),
	}
	return values, diagnostics
}

// objectClasses preserves AD DS's ordered objectClass sequence; the most specific instantiated class is last.
// See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-adts/1de0b9e3-6cc6-4a8c-b9da-715dc301645b.
func objectClasses(object shared.Object) (types.List, diag.Diagnostics) {
	values := object.Values("objectClass")
	var diagnostics diag.Diagnostics
	classes := make([]attr.Value, 0, len(values.Values))
	if values.State != shared.AttributePresent || len(values.Values) == 0 {
		diagnostics.AddWarning(
			"Object classes unavailable",
			"A complete ordered objectClass sequence was not observed. The class remains null; usable identity and other observations are retained.",
		)
		return types.ListNull(types.StringType), diagnostics
	}
	for _, value := range values.Values {
		text, err := value.Text()
		if err != nil || !attributeName.MatchString(text) {
			diagnostics.AddWarning("Invalid object class observation", "The returned class sequence contains an unusable LDAP class name; no class is guessed.")
			return types.ListNull(types.StringType), diagnostics
		}
		classes = append(classes, types.StringValue(text))
	}
	if diagnostics.HasError() {
		return types.ListNull(types.StringType), diagnostics
	}
	return types.ListValueMust(types.StringType, classes), diagnostics
}

// singleText refuses unloaded, partial and multivalued identity fields without inferring a value from a DN spelling.
func singleText(object shared.Object, name string) types.String {
	value, err := object.Values(name).Single()
	if err != nil {
		return types.StringNull()
	}
	text, err := value.Text()
	if err != nil || text == "" {
		return types.StringNull()
	}
	return types.StringValue(text)
}

// rawAttributes exposes omission as a nullable observation, never as proof of a successful attribute clear.
func rawAttributes(object shared.Object, properties []string) (map[string]attr.Value, map[string]attr.Value, map[string]attr.Value) {
	attributes, status, ranges := map[string]attr.Value{}, map[string]attr.Value{}, map[string]attr.Value{}
	for _, property := range properties {
		name := strings.ToLower(property)
		if !writableAttribute(name) {
			continue
		}
		values := object.Values(name)
		attributes[name] = types.SetNull(types.StringType)
		switch values.State {
		case shared.AttributePresent:
			attributes[name] = encodedSet(values.Values)
			status[name] = types.StringValue("present")
		case shared.AttributeAbsent:
			status[name] = types.StringValue("absent")
		case shared.AttributePartial:
			status[name] = types.StringValue("partial")
			for description, raw := range values.Ranges {
				ranges[description] = encodedSet(raw)
			}
		default:
			status[name] = types.StringValue("unloaded")
		}
	}
	return attributes, status, ranges
}

// encodedSet converts only at the explicit Terraform set boundary, retaining binary and zero-length values.
func encodedSet(raw []shared.RawValue) types.Set {
	encoded := EncodeValues(raw)
	values := make([]attr.Value, 0, len(encoded))
	for _, value := range encoded {
		values = append(values, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, values)
}
