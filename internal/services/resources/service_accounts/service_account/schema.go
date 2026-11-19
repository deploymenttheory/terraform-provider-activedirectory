// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_account

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
)

// serviceAccountSchema separates directory metadata, explicitly owned reader grants and creation-only inputs.
func serviceAccountSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable nonzero directory object GUID used for lifecycle and import.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":      schema.StringAttribute{Required: true, Description: "Literal directory name. Changes rename the same GUID."},
		"parent_dn": schema.StringAttribute{Required: true, Description: "Explicit parent DN. Changes move the same GUID within its domain."},
		"kind": schema.StringAttribute{
			Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Description: "standalone (sMSA) or group_managed (gMSA). Changing kind replaces the account. Ordinary dMSA creation is unavailable.",
		},
		"sam_account_name":   schema.StringAttribute{Required: true, Description: "Explicit canonical account name with exactly one trailing $."},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"sid":                schema.StringAttribute{Computed: true, Description: "Directory-assigned account SID when visible."},
		"recovery_dn":        schema.StringAttribute{Computed: true, Description: "Recovery locator if creation acceptance precedes GUID observation."},
		"outbound_authentication_only": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(false),
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			Description: "Creation-only gMSA outbound form. Conflicts with DNS host name, password interval, reader grants and compound identity inputs. " +
				"This is stored creation intent, not a persistent server restriction. Import initializes it to false.",
		},
		"managed_password_interval_days": schema.Int64Attribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown(), int64planmodifier.RequiresReplace()},
			Description:   "Creation-only gMSA managed password interval from 1 to 2147483647 days; the SDK defaults to 30. Unavailable for sMSAs.",
		},
		"account_expires": schema.Int64Attribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description: "Nonnegative FILETIME. Zero and 9223372036854775807 mean never. Omission retains the directory value.",
		},
		"compound_identity_supported": schema.BoolAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			Description: "Explicit compound identity bit, managed independently of encryption policy and unrelated bits.",
		},
		"password_wo": schema.StringAttribute{
			Optional: true, Sensitive: true, WriteOnly: true,
			Description: "Creation-only UTF-8 sMSA password from an ephemeral variable; requires a positive password_wo_version. " +
				"When omitted, the SDK generates and erases a strong random sMSA password. Never valid for gMSAs or returned in state.",
		},
		"password_wo_version": schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown(), int64planmodifier.RequiresReplace()},
			Description:   "Acknowledged explicit creation password version. Changing it replaces the account; this resource never rotates a managed password. Import starts at zero.",
		},
		"clear_properties": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Explicit optional LDAP text properties to clear on an existing object. Identity, passwords and raw flag masks are excluded.",
		},
	}
	for name, description := range map[string]string{
		"kerberos_encryption_types": "Explicit des_crc, des_md5, rc4, aes128 or aes256 account-policy bits. An empty set selects zero low bits; unrelated flags are preserved.",
		"service_principal_names":   "Owned complete SPN values. An empty set requests clearing; omitted or incomplete observations never imply an empty set.",
		"password_reader_sids": "gMSA principal SIDs owning explicit ordinary Read Property allow ACEs only. " +
			"Empty removes those grants; deny, conditional, inherited ACEs and unrelated rights are preserved. This is not effective-access evaluation.",
	} {
		attributes[name] = schema.SetAttribute{
			Optional: true, Computed: true, ElementType: types.StringType,
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, Description: description,
		}
	}
	for name, ldap := range map[string]string{
		"dns_host_name": "dNSHostName", "display_name": "displayName", "description": "description", "home_page": "wWWHomePage",
	} {
		attributes[name] = schema.StringAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description: "Typed " + ldap + "; omission retains the directory observation without acquiring mutation intent.",
		}
	}
	model, control := ServiceAccountResourceModel{}, accounts.AccountControlPatch{}
	for _, field := range boolProperties(&model, &control) {
		attributes[field.name] = schema.BoolAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			Description: "Explicit " + field.name + " stored account setting. Unconfigured observations never implicitly authorize a change.",
		}
	}
	return schema.Schema{
		Description: "Manage one standalone or group-managed AD DS service account by GUID. " +
			"Directory prerequisites are checked by the SDK. No host installation, password retrieval or implicit computer association occurs. Deletion is leaf-only.",
		Attributes: attributes,
	}
}
