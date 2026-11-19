// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// DataSourceAttributes describes observable OU fields without treating hidden attributes as false/empty defaults.
func DataSourceAttributes() map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{}
	for name := range AttributeTypes() {
		if name == "protected_from_accidental_deletion" {
			attributes[name] = schema.BoolAttribute{Computed: true, Description: "Observed deletion protection, or null when unavailable."}
		} else {
			attributes[name] = schema.StringAttribute{Computed: true, Description: "Observed " + name + "; unavailable optional values remain null."}
		}
	}
	return attributes
}
