// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/memberships"
)

// mapState preserves explicit input spelling and emits the common canonical import identity and observed names.
func mapState(ctx context.Context, state *tfsdk.State, model GroupMemberDataSourceModel, edge common.Edge) diag.Diagnostics {
	values, err := edge.Values()
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Incomplete membership identity", "No usable direct-link identity was returned.")}
	}
	values["group_guid"], values["member_guid"] = model.GroupGUID, model.MemberGUID
	var diagnostics diag.Diagnostics
	for name, value := range values {
		diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diagnostics
}
