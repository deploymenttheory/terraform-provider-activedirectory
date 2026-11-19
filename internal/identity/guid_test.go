// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package identity

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const objectID = "a3e9767a-200e-43dd-bb1d-e5ab400e5d6f"
const memberID = "b3e9767a-200e-43dd-bb1d-e5ab400e5d6f"

func TestGUIDImportAndRelationshipIdentity(t *testing.T) {
	ctx := context.Background()
	for _, value := range []string{objectID, "{A3E9767A-200E-43DD-BB1D-E5AB400E5D6F}"} {
		response := resource.ImportStateResponse{State: tfsdk.State{
			Schema: schema.Schema{Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}}},
			Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}}, map[string]tftypes.Value{
				"id": tftypes.NewValue(tftypes.String, nil),
			}),
		}}
		ImportGUID(ctx, resource.ImportStateRequest{ID: value}, &response)
		var id types.String
		response.Diagnostics.Append(response.State.GetAttribute(ctx, path.Root("id"), &id)...)
		if response.Diagnostics.HasError() || id.ValueString() != objectID {
			t.Fatalf("import failed: %v %v", id, response.Diagnostics)
		}
	}
	edge, err := RelationshipID(objectID, memberID)
	if err != nil {
		t.Fatal(err)
	}
	owner, member, diagnostics := ParseRelationshipID(edge)
	if diagnostics.HasError() || owner != objectID || member != memberID {
		t.Fatal("relationship endpoints did not round trip in directed order")
	}
	for _, value := range []string{"", "CN=example,DC=invalid", "guid:" + objectID, "00000000-0000-0000-0000-000000000000", objectID + "/" + memberID} {
		if _, err := CanonicalGUID(value); err == nil {
			t.Errorf("invalid object identity accepted: %q", value)
		}
	}
	for _, value := range []string{objectID, edge + "/extra", objectID + "/CN=member,DC=invalid"} {
		if _, _, diagnostics := ParseRelationshipID(value); !diagnostics.HasError() {
			t.Errorf("invalid edge identity accepted: %q", value)
		}
	}
}
