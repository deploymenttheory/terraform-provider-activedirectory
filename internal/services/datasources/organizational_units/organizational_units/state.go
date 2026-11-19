// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/organizational_units"
	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"
)

// mapState preserves usable partial rows and deterministic ordering without hiding invalid or duplicate identity.
func mapState(
	ctx context.Context,
	state *tfsdk.State,
	model OrganizationalUnitsDataSourceModel,
	options shared.SearchOptions,
	result organizational_units.OrganizationalUnitsResult,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	objects := map[string]attr.Value{}
	for _, unit := range result.Units {
		values, projection := common.Project(unit)
		diagnostics.Append(projection...)
		if projection.HasError() {
			continue
		}
		id := unit.GUID.String()
		if _, duplicate := objects[id]; duplicate {
			diagnostics.AddError("Duplicate OU identity", "The search returned the same GUID more than once; discovery cannot be treated as complete.")
			continue
		}
		value, conversion := types.ObjectValue(common.AttributeTypes(), values)
		diagnostics.Append(conversion...)
		objects[id] = value
	}
	ids := make([]string, 0, len(objects))
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, objects[id])
	}
	list, conversion := types.ListValue(types.ObjectType{AttrTypes: common.AttributeTypes()}, rows)
	diagnostics.Append(conversion...)
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d", options.Base, options.Filter, options.Scope, options.PageSize, options.ResultLimit)))
	for name, value := range map[string]any{
		"id": fmt.Sprintf("%x", fingerprint), "base_dn": model.BaseDN, "ldap_filter": model.LDAPFilter, "scope": model.Scope,
		"page_size": model.PageSize, "result_limit": model.ResultLimit, "organizational_units": list,
		"complete": result.Observation.Complete && !diagnostics.HasError(), "limit_reached": result.Observation.LimitReached,
	} {
		diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diagnostics
}
