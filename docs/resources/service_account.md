---
page_title: "activedirectory_service_account Resource - Active Directory"
subcategory: "Service Accounts"
description: |-
  Manage a standalone or group-managed directory service account by GUID.
---

# activedirectory_service_account (Resource)

Manage one standalone or group-managed AD DS service account by GUID. Directory prerequisites are checked by the SDK. No host installation, password retrieval or implicit computer association occurs. Deletion is leaf-only.

```terraform
terraform {
  required_version = ">= 1.14.0"
  required_providers {
    activedirectory = { source = "deploymenttheory/activedirectory" }
  }
}

# Supply AD_HOST, AD_USERNAME, AD_PASSWORD and, for a private CA, AD_CA_CERT_PEM
# through your secret manager or process environment. Hostname/chain checks remain enabled.
provider "activedirectory" {
  auth_method = "simple"
  transport   = "ldaps"
}
```

```terraform
resource "activedirectory_service_account" "example" {
  name             = "Terraform Build"
  sam_account_name = "tfbuild$"
  parent_dn        = "OU=Service Accounts,DC=example,DC=test"
  kind             = "group_managed"
  dns_host_name    = "tfbuild.example.test"
  enabled          = false
}
```

```terraform
variable "password_reader_sids" {
  type        = set(string)
  description = "Canonical SIDs of principals allowed explicit gMSA Read Property grants."
}

resource "activedirectory_service_account" "extended" {
  name                                 = "Terraform Extended Build"
  sam_account_name                     = "tfextended$"
  parent_dn                            = "OU=Service Accounts,DC=example,DC=test"
  kind                                 = "group_managed"
  dns_host_name                        = "tfextended.example.test"
  description                          = "Terraform-owned build identity"
  display_name                         = "Extended build service"
  home_page                            = "https://example.test/build"
  enabled                              = false
  managed_password_interval_days       = 30
  service_principal_names              = ["HTTP/tfextended.example.test"]
  password_reader_sids                 = var.password_reader_sids
  kerberos_encryption_types            = ["aes128", "aes256"]
  compound_identity_supported          = true
  account_expires                      = 0
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_pre_auth            = false
  home_directory_required              = false
  mns_logon_account                    = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
}
```

The parent DN and canonical sAMAccountName ending in one `$` are explicit. Supported kinds
are `standalone` (sMSA) and `group_managed` (gMSA). The SDK checks the selected writable AD DS
target, schema and applicable KDS prerequisites before creation. This resource does not
provision KDS keys, install accounts on hosts or create computer associations. Ordinary dMSA
creation is unavailable. Live provider qualification of this feature remains in progress.

Name and same-domain parent changes retain the GUID. Kind, managed password interval,
outbound creation form and creation-password version changes require replacement. Typed
settings preserve unrelated account/encryption bits. Optional attributes remain unmanaged
when omitted; use `clear_properties` for explicit optional text clearing on an existing account.
An empty SPN set requests clearing. Unavailable/partial observations retain prior state and
never prove a clear or an empty collection. Do not manage the same GUID/attributes with a
generic object or another typed resource.

`password_reader_sids` owns explicit ordinary Read Property allow ACEs for a gMSA. Canonical
principal SIDs are resolved through the SDK. Empty removes these grants; inherited, deny,
conditional ACEs, unrelated masks and other descriptor components are preserved. A returned
policy is not an effective-access guarantee. Hidden descriptors cannot authorize policy edits.

## Standalone creation passwords

When an sMSA password is omitted, the SDK generates and erases a strong random password.
For an explicit initial password, use an ephemeral variable and a positive creation version:

```terraform
variable "standalone_initial_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "activedirectory_service_account" "standalone" {
  name                = "Standalone Build"
  sam_account_name    = "tfstandalone$"
  parent_dn           = "OU=Service Accounts,DC=example,DC=test"
  kind                = "standalone"
  enabled             = false
  password_wo         = var.standalone_initial_password
  password_wo_version = 1
}
```

`password_wo` never enters Terraform plan/state. Its version is nonsecret acknowledgement
metadata and is creation-only: changing it replaces the account, rather than resetting a
password in place. gMSA passwords are generated by the DC and never retrieved. If `enabled`
is omitted during creation, the SDK follows the default enabled creation workflow; set it
explicitly when a disabled account is required.

## Outbound-only gMSA creation

```terraform
resource "activedirectory_service_account" "outbound" {
  name                         = "Outbound Build"
  sam_account_name             = "tfoutbound$"
  parent_dn                    = "OU=Service Accounts,DC=example,DC=test"
  kind                         = "group_managed"
  outbound_authentication_only = true
  enabled                      = false
}
```

This creation form excludes DNS host name, managed password interval, reader grants and
compound identity inputs. It records how the object was created; it is not a persistent
server flag that constrains later directory updates. Import initializes this local intent
to false because the original creation form cannot be discovered.

## Failure recovery and deletion

Accepted stages retain their GUID and acknowledgement metadata if password setup, enablement
or readback fails. Uncertain submissions retain a private checkpoint and block another write,
including tainted replacement. Do not retry creation or delete a possible existing object.
Use `recovery_dn` when the GUID was unavailable, independently establish the existing GUID
and accepted stages, then explicitly import that verified identity after reconciliation.
No password readback, login probe, automatic rollback or mutation replay occurs.

Deletion is leaf-only. Protected, nonempty, denied or uncertain deletion retains state;
positive same-GUID tombstone evidence or pure confirmed absence is required for removal.

## Import

```terraform
import {
  to = activedirectory_service_account.example
  id = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
}
```

```terraform
import {
  to = activedirectory_service_account.example
  identity = {
    id = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
  }
}
```

Import reads the existing account without password material and starts `password_wo_version`
at zero. Original outbound creation intent is not inferred. Explicit clear examples are in
`examples/resources/activedirectory_service_account/clear/`; import the existing account first.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `kind` (String) standalone (sMSA) or group_managed (gMSA). Changing kind replaces the account. Ordinary dMSA creation is unavailable.
- `name` (String) Literal directory name. Changes rename the same GUID.
- `parent_dn` (String) Explicit parent DN. Changes move the same GUID within its domain.
- `sam_account_name` (String) Explicit canonical account name with exactly one trailing $.

### Optional

> **NOTE**: [Write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments) are supported in Terraform 1.11 and later.

- `account_expires` (Number) Nonnegative FILETIME. Zero and 9223372036854775807 mean never. Omission retains the directory value.
- `account_not_delegated` (Boolean) Explicit account_not_delegated stored account setting. Unconfigured observations never implicitly authorize a change.
- `allow_reversible_password_encryption` (Boolean) Explicit allow_reversible_password_encryption stored account setting. Unconfigured observations never implicitly authorize a change.
- `clear_properties` (Set of String) Explicit optional LDAP text properties to clear on an existing object. Identity, passwords and raw flag masks are excluded.
- `compound_identity_supported` (Boolean) Explicit compound identity bit, managed independently of encryption policy and unrelated bits.
- `description` (String) Typed description; omission retains the directory observation without acquiring mutation intent.
- `display_name` (String) Typed displayName; omission retains the directory observation without acquiring mutation intent.
- `dns_host_name` (String) Typed dNSHostName; omission retains the directory observation without acquiring mutation intent.
- `does_not_require_pre_auth` (Boolean) Explicit does_not_require_pre_auth stored account setting. Unconfigured observations never implicitly authorize a change.
- `enabled` (Boolean) Explicit enabled stored account setting. Unconfigured observations never implicitly authorize a change.
- `home_directory_required` (Boolean) Explicit home_directory_required stored account setting. Unconfigured observations never implicitly authorize a change.
- `home_page` (String) Typed wWWHomePage; omission retains the directory observation without acquiring mutation intent.
- `kerberos_encryption_types` (Set of String) Explicit des_crc, des_md5, rc4, aes128 or aes256 account-policy bits. An empty set selects zero low bits; unrelated flags are preserved.
- `managed_password_interval_days` (Number) Creation-only gMSA managed password interval from 1 to 2147483647 days; the SDK defaults to 30. Unavailable for sMSAs.
- `mns_logon_account` (Boolean) Explicit mns_logon_account stored account setting. Unconfigured observations never implicitly authorize a change.
- `outbound_authentication_only` (Boolean) Creation-only gMSA outbound form. Conflicts with DNS host name, password interval, reader grants and compound identity inputs. This is stored creation intent, not a persistent server restriction. Import initializes it to false.
- `password_reader_sids` (Set of String) gMSA principal SIDs owning explicit ordinary Read Property allow ACEs only. Empty removes those grants; deny, conditional, inherited ACEs and unrelated rights are preserved. This is not effective-access evaluation.
- `password_wo` (String, Sensitive, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) Creation-only UTF-8 sMSA password from an ephemeral variable; requires a positive password_wo_version. When omitted, the SDK generates and erases a strong random sMSA password. Never valid for gMSAs or returned in state.
- `password_wo_version` (Number) Acknowledged explicit creation password version. Changing it replaces the account; this resource never rotates a managed password. Import starts at zero.
- `service_principal_names` (Set of String) Owned complete SPN values. An empty set requests clearing; omitted or incomplete observations never imply an empty set.
- `smartcard_logon_required` (Boolean) Explicit smartcard_logon_required stored account setting. Unconfigured observations never implicitly authorize a change.
- `trusted_for_delegation` (Boolean) Explicit trusted_for_delegation stored account setting. Unconfigured observations never implicitly authorize a change.
- `trusted_to_auth_for_delegation` (Boolean) Explicit trusted_to_auth_for_delegation stored account setting. Unconfigured observations never implicitly authorize a change.
- `use_des_key_only` (Boolean) Explicit use_des_key_only stored account setting. Unconfigured observations never implicitly authorize a change.

### Read-Only

- `distinguished_name` (String) Current directory-observed distinguished name.
- `id` (String) Stable nonzero directory object GUID used for lifecycle and import.
- `recovery_dn` (String) Recovery locator if creation acceptance precedes GUID observation.
- `sid` (String) Directory-assigned account SID when visible.
