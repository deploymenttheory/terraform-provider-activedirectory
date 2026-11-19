// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computer

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/accounts"
)

// computerSchema describes directory ownership and write-only password intent without exposing domain-join or host-management operations.
func computerSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable computer object GUID used for lifecycle and import identity.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":      schema.StringAttribute{Required: true, Description: "Literal directory object name. Changes rename the same GUID."},
		"parent_dn": schema.StringAttribute{Required: true, Description: "Explicit parent DN; changing it moves the object within its domain."},
		"sam_account_name": schema.StringAttribute{
			Required: true, Description: "Explicit canonical computer sAMAccountName with one trailing $. No account name or container is inferred.",
		},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"sid":                schema.StringAttribute{Computed: true, Description: "Directory-assigned SID when visible."},
		"recovery_dn":        schema.StringAttribute{Computed: true, Description: "Recovery locator when creation acceptance precedes GUID observation."},
		"account_expires": schema.Int64Attribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description: "Raw nonnegative FILETIME. Zero and 9223372036854775807 mean never. Omission retains the directory value.",
		},
		"kerberos_encryption_types": schema.SetAttribute{
			Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			Description: "Explicit des_crc, des_md5, rc4, aes128 or aes256 account policy bits. " +
				"Omission retains observed values; an empty set selects zero low bits. Compound identity and unrelated bits remain independent.",
		},
		"compound_identity_supported": schema.BoolAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			Description: "Owns the account's compound identity bit independently of encryption selection and unrelated flags.",
		},
		"password_wo": schema.StringAttribute{
			Optional: true, Sensitive: true, WriteOnly: true,
			Description: "Write-only UTF-8 account password supplied through an ephemeral variable. " +
				"Requires an explicit positive version and never appears in plan/state. This does not join a host to the domain.",
		},
		"password_wo_version": schema.Int64Attribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description: "Acknowledged password intent version. Increase with password_wo to request a reset; decreasing is refused. Import starts at zero.",
		},
		"clear_properties": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Explicit optional LDAP text properties to clear on an existing object. Excludes account identity, passwords, expiration and raw flag masks.",
		},
	}
	for name, ldap := range map[string]string{
		"dns_host_name": "dNSHostName", "user_principal_name": "userPrincipalName", "display_name": "displayName", "description": "description",
		"location": "location", "managed_by_dn": "managedBy", "home_page": "wWWHomePage", "operating_system": "operatingSystem",
		"operating_system_version": "operatingSystemVersion", "operating_system_service_pack": "operatingSystemServicePack", "operating_system_hotfix": "operatingSystemHotfix",
	} {
		attributes[name] = schema.StringAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description: "Typed " + ldap + " directory property. Omission retains its observed value. Metadata does not configure the corresponding host.",
		}
	}
	model, control := ComputerResourceModel{}, accounts.AccountControlPatch{}
	for _, field := range boolProperties(&model, &control) {
		definition := schema.BoolAttribute{
			Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			Description: "Explicit " + field.name + " account setting; omitted values retain their observations and are never implicitly enabled.",
		}
		if field.name == "change_password_at_logon" {
			// Password resets can change pwdLastSet independently. Leaving this
			// computed field unknown lets refresh report that effect without
			// replaying a prior unconfigured observation as owned intent.
			definition.PlanModifiers = nil
			definition.Description = "Explicit pwdLastSet semantics. On creation, applied after password setup in a separate typed phase. " +
				"When omitted, observation can change after password reset without acquiring ownership."
		}
		attributes[field.name] = definition
	}
	return schema.Schema{
		Description: "Manage one AD DS computer object by GUID with staged creation and selected typed settings. " +
			"This resource never joins or configures a machine. Deletion is leaf-only.",
		Attributes: attributes,
	}
}
