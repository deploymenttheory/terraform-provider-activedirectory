// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package object

import "github.com/hashicorp/terraform-plugin-framework/types"

// ObjectResourceModel separates configured raw intent from directory observations and stable GUID identity.
type ObjectResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	ParentDN           types.String `tfsdk:"parent_dn"`
	ObjectClass        types.String `tfsdk:"object_class"`
	DistinguishedName  types.String `tfsdk:"distinguished_name"`
	ObjectClasses      types.List   `tfsdk:"object_classes"`
	Attributes         types.Map    `tfsdk:"attributes"`
	AddValues          types.Map    `tfsdk:"add_values"`
	RemoveValues       types.Map    `tfsdk:"remove_values"`
	ClearAttributes    types.Set    `tfsdk:"clear_attributes"`
	Properties         types.Set    `tfsdk:"properties"`
	ObservedAttributes types.Map    `tfsdk:"observed_attributes"`
	AttributeStatus    types.Map    `tfsdk:"attribute_status"`
	AttributeRanges    types.Map    `tfsdk:"attribute_ranges"`
	NeedsUpdate        types.Bool   `tfsdk:"needs_update"`
	RecoveryDN         types.String `tfsdk:"recovery_dn"`
}
