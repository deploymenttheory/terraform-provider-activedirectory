// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computers

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceAttributes preserves unavailable observations as typed null values.
func DataSourceAttributes() map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{}
	for name, kind := range AttributeTypes() {
		switch {
		case kind.Equal(types.BoolType):
			attributes[name] = schema.BoolAttribute{Computed: true, Description: "Observed account flag; null when unavailable."}
		case kind.Equal(types.Int64Type):
			attributes[name] = schema.Int64Attribute{Computed: true, Description: "Observed raw FILETIME; null when unavailable."}
		case kind.Equal(types.SetType{ElemType: types.StringType}):
			attributes[name] = schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Observed named encryption bits; null when unavailable."}
		default:
			attributes[name] = schema.StringAttribute{Computed: true, Description: "Observed " + name + "; unavailable optional values remain null."}
		}
	}
	return attributes
}
