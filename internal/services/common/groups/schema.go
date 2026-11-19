// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package groups

import "github.com/hashicorp/terraform-plugin-framework/datasource/schema"

// DataSourceAttributes preserves unavailable directory properties as nullable observations, not fabricated defaults.
func DataSourceAttributes() map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{}
	for name := range AttributeTypes() {
		attributes[name] = schema.StringAttribute{Computed: true, Description: "Observed " + name + "; unavailable values remain null."}
	}
	return attributes
}
