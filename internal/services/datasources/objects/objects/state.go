// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"
	common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"
)

// mapState preserves usable partial rows and deterministic ordering without hiding invalid or duplicate identity.
func mapState(
	ctx context.Context,
	state *tfsdk.State,
	model ObjectsDataSourceModel,
	options shared.SearchOptions,
	result shared.SearchResult,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	objects := map[string]attr.Value{}
	for _, unit := range result.Objects {
		values, projection := common.Project(unit, options.Properties)
		diagnostics.Append(projection...)
		if projection.HasError() {
			continue
		}
		id := values["id"].(types.String).ValueString()
		if _, duplicate := objects[id]; duplicate {
			diagnostics.AddError("Duplicate object identity", "The search returned the same GUID more than once; discovery cannot be treated as complete.")
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
	if len(ids) > options.ResultLimit {
		ids = ids[:options.ResultLimit]
		result.Complete = false
		result.LimitReached = true
	}
	rows := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, objects[id])
	}
	list, conversion := types.ListValue(types.ObjectType{AttrTypes: common.AttributeTypes()}, rows)
	diagnostics.Append(conversion...)
	fingerprint := sha256.Sum256(
		[]byte(
			fmt.Sprintf(
				"%s\x00%s\x00%d\x00%d\x00%d\x00%s",
				options.Base,
				options.Filter,
				options.Scope,
				options.PageSize,
				options.ResultLimit,
				strings.Join(options.Properties, ","),
			),
		),
	)
	for name, value := range map[string]any{
		"id": fmt.Sprintf("%x", fingerprint), "base_dn": model.BaseDN, "ldap_filter": model.LDAPFilter, "scope": model.Scope,
		"properties": model.Properties, "page_size": model.PageSize, "result_limit": model.ResultLimit, "objects": list,
		"complete": result.Complete && !result.LimitReached && !diagnostics.HasError(), "limit_reached": result.LimitReached,
	} {
		diagnostics.Append(state.SetAttribute(ctx, path.Root(name), value)...)
	}
	return diagnostics
}
