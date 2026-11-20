// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_members

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Schema requires an existing group GUID and exposes independent traversal limits and completeness observations.
func (*GroupMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id":         schema.StringAttribute{Computed: true, Description: "Canonical GUID of the group selected by this query."},
		"group_guid": schema.StringAttribute{Required: true, Description: "Existing group object GUID; names and DNs are not query identities."},
		"group_dn":   schema.StringAttribute{Computed: true, Description: "Explicitly observed current group distinguished name."},
		"recursive": schema.BoolAttribute{
			Optional: true,
			Description: "Traverse nested groups and return leaf principals when true. Defaults to false. " +
				"Primary-group membership is included, matching Get-ADGroupMember.",
		},
		"complete": schema.BoolAttribute{
			Computed: true,
			Description: "Whether requested traversal, ranges and identities completed without known omission or failure. " +
				"This is not an effective-access or transactional graph guarantee.",
		},
		"limit_reached":  schema.BoolAttribute{Computed: true, Description: "Whether an aggregate traversal budget stopped retrieval; further matches may or may not exist."},
		"groups_visited": schema.Int64Attribute{Computed: true, Description: "Number of group traversal observations retained by the SDK."},
		"visibility_known": schema.BoolAttribute{
			Computed: true,
			Description: "Whether every visited group returned at least one direct member value. " +
				"False preserves the ambiguity shared by an empty and hidden attribute, even when primary members were found.",
		},
		"members": schema.ListNestedAttribute{
			Computed: true, Description: "Usable principals sorted by source host, port and GUID. Duplicate observations merge only when identity and names agree.",
			NestedObject: schema.NestedAttributeObject{Attributes: memberAttributes()},
		},
		"unresolved": schema.ListNestedAttribute{
			Computed:    true,
			Description: "Bounded references the SDK could not resolve in the explicitly authenticated context. No foreign connection or credential forwarding is inferred.",
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"reference":   schema.StringAttribute{Computed: true, Description: "Returned directory reference; it is not a resolved principal GUID."},
				"source_host": schema.StringAttribute{Computed: true, Description: "SDK source host, when observed."},
			}},
		},
	}
	for name, description := range map[string]string{
		"max_objects": "Aggregate object observation budget from 1 to 10000, including group/resolution observations; " +
			"defaults to 1000.",
		"max_groups": "Maximum traversed groups from 1 to 1000; defaults to 100.",
		"max_ranges": "Aggregate member-range read budget from 1 to 1000; defaults to 100.",
		"max_depth":  "Maximum nested traversal depth from 1 to 32; defaults to 8. Direct mode does not expand nested groups.",
		"max_bytes":  "Aggregate observed attribute/name byte budget from 1024 to 67108864; defaults to 8388608.",
	} {
		attributes[name] = schema.Int64Attribute{Optional: true, Description: description}
	}
	resp.Schema = schema.Schema{
		Description: "Read bounded direct or explicitly recursive group membership, including primary-group relationships. This data source changes no membership or directory objects.",
		Attributes:  attributes,
	}
}

// memberAttributes describes observed principal identity without making the query an import or ownership surface.
func memberAttributes() map[string]schema.Attribute {
	attributes := map[string]schema.Attribute{}
	for name, description := range map[string]string{
		"id": "Observed principal GUID.", "sid": "Observed security identifier.",
		"distinguished_name": "Explicitly returned current distinguished name.", "name": "Returned directory name, when available.",
		"sam_account_name": "Returned account name, when available.", "object_class": "Most specific returned directory class.",
		"source_host": "SDK source host for this principal.",
	} {
		attributes[name] = schema.StringAttribute{Computed: true, Description: description}
	}
	attributes["source_port"] = schema.Int64Attribute{Computed: true, Description: "SDK source port for this principal."}
	attributes["primary"] = schema.BoolAttribute{Computed: true, Description: "Whether any observed path to this principal included a primary-group edge."}
	return attributes
}
