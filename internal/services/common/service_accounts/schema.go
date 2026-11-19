// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_accounts

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
			attributes[name] = schema.Int64Attribute{Computed: true, Description: "Observed integer; null when unavailable."}
		case kind.Equal(types.SetType{ElemType: types.StringType}):
			attributes[name] = schema.SetAttribute{
				Computed: true, ElementType: types.StringType,
				Description: "Observed values; null when unavailable. This does not establish effective access.",
			}
		default:
			attributes[name] = schema.StringAttribute{Computed: true, Description: "Observed " + name + "; unavailable optional values remain null."}
		}
	}
	return attributes
}
