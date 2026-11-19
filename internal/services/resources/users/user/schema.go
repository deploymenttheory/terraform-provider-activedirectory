// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package user

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/services/users"
)

// userSchema describes property ownership independently of CRUD execution and keeps the secret write-only.
func userSchema() schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true, Description: "Stable user object GUID and import identity.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":             schema.StringAttribute{Required: true, Description: "Literal common name. Changes rename the same object GUID."},
		"parent_dn":        schema.StringAttribute{Required: true, Description: "Explicit parent DN. Changes move the same object within its domain."},
		"sam_account_name": schema.StringAttribute{Required: true, Description: "Explicit sAMAccountName; no name or container is guessed."},
		"object_class": schema.StringAttribute{
			Optional: true, Computed: true,
			Description:   "Creation class: user (default) or inetOrgPerson. Changing class replaces the resource. Omitted updates retain the observed class.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
		},
		"distinguished_name": schema.StringAttribute{Computed: true, Description: "Current directory-observed distinguished name."},
		"sid":                schema.StringAttribute{Computed: true, Description: "Directory-assigned security identifier when visible."},
		"recovery_dn":        schema.StringAttribute{Computed: true, Description: "Recovery locator retained when creation acceptance precedes GUID observation."},
		"password_wo": schema.StringAttribute{
			Optional: true, Sensitive: true, WriteOnly: true,
			Description: "Write-only UTF-8 password. Requires a positive password_wo_version. Never persisted in plan or state; omitted input is not recovered from prior state.",
		},
		"password_wo_version": schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description: "Acknowledged password intent version. Increase with password_wo to request a reset; decreasing is refused. " +
				"Omission retains the last version. Import starts at zero.",
		},
		"account_expires": schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Description: "Raw nonnegative FILETIME expiration. Zero and 9223372036854775807 explicitly mean never. " +
				"Omission retains the observed value; no local date parsing is performed.",
		},
		"clear_properties": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Explicit LDAP text attributes to clear on an existing user. Excludes sAMAccountName, accountExpires, passwords and account-control masks.",
		},
	}
	model, properties := UserResourceModel{}, users.UserProperties{}
	for _, field := range textProperties(&model, &properties) {
		if field.terraform == "sam_account_name" {
			continue
		}
		attributes[field.terraform] = schema.StringAttribute{
			Optional: true, Computed: true,
			Description:   "Typed " + field.ldap + " property. Omit to retain the observed value; use clear_properties for an explicit clear.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	for name, description := range map[string]string{
		"enabled":                  "Whether the account is enabled. Creation starts disabled, sets an initial password when supplied, and enables only when explicitly requested.",
		"password_not_required":    "Explicit password-not-required account flag. It is never enabled implicitly to bypass initial password setup.",
		"password_never_expires":   "Stored password-never-expires account flag.",
		"smartcard_logon_required": "Stored smartcard-logon-required account flag.",
	} {
		attributes[name] = schema.BoolAttribute{
			Optional: true, Computed: true, Description: description + " Omitted updates retain the observed flag.",
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	return schema.Schema{
		Description: "Manage one AD DS user by GUID with staged creation, explicitly owned properties and versioned write-only password changes. Deletion is leaf-only.",
		Attributes:  attributes,
	}
}
