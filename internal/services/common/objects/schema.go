// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceAttributes exposes identity and explicit raw observations without taking directory ownership.
func DataSourceAttributes() map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{}
	for _, name := range []string{"id", "name", "parent_dn", "distinguished_name", "object_class"} {
		attributes[name] = schema.StringAttribute{Computed: true, Description: "Directory-observed " + name + "."}
	}
	attributes["object_classes"] = schema.ListAttribute{
		Computed: true, ElementType: types.StringType,
		Description: "Ordered AD DS objectClass sequence; the instantiated class is last.",
	}
	attributes["observed_attributes"] = schema.MapAttribute{
		Computed: true, ElementType: RawSetType(),
		Description: "Explicitly selected raw attributes as canonical padded base64 sets. Unavailable values remain null; omission does not prove an attribute is unset.",
	}
	attributes["attribute_status"] = schema.MapAttribute{
		Computed: true, ElementType: types.StringType,
		Description: "Retrieval status for each selected raw attribute: present, absent, unloaded or partial. Absent can mean access-hidden.",
	}
	attributes["attribute_ranges"] = schema.MapAttribute{
		Computed: true, ElementType: RawSetType(),
		Description: "Raw returned range descriptions and base64 sets. These are incomplete ranges, never a complete attribute value set.",
	}
	return attributes
}
