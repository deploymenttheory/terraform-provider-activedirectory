---
page_title: "activedirectory_computer Resource - Active Directory"
subcategory: "Computers"
description: |-
  Manage a computer by stable GUID with staged creation and write-only password input.
---

# activedirectory_computer (Resource)

Manage one AD DS computer object by GUID with staged creation and selected typed settings. This resource never joins or configures a machine. Deletion is leaf-only.

## Provider configuration

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

## Disabled computer

```terraform
resource "activedirectory_computer" "computer" {
  name             = "terraform-computer"
  sam_account_name = "terraform-computer$"
  parent_dn        = "OU=Automation,DC=example,DC=test"
  enabled          = false
}
```

## Password setup and explicit enablement

```terraform
variable "computer_password" {
  type        = string
  sensitive   = true
  ephemeral   = true
  description = "Initial or explicitly rotated password from your secret manager. Never persist it in configuration files."
}

resource "activedirectory_computer" "full" {
  name                                 = "tf-machine"
  sam_account_name                     = "tf-machine$"
  parent_dn                            = "OU=Automation,DC=example,DC=test"
  dns_host_name                        = "tf-machine.example.test"
  user_principal_name                  = "tf-machine$@example.test"
  display_name                         = "Terraform managed machine account"
  description                          = "Directory account metadata only; no host join or configuration"
  location                             = "Lab rack 1"
  managed_by_dn                        = "CN=Automation Owner,OU=People,DC=example,DC=test"
  home_page                            = "https://inventory.example.test/machines/tf-machine"
  operating_system                     = "Windows Server 2025"
  operating_system_version             = "10.0"
  operating_system_service_pack        = "Inventory metadata"
  operating_system_hotfix              = "Inventory metadata"
  account_expires                      = 0
  kerberos_encryption_types            = ["aes128", "aes256"]
  compound_identity_supported          = true
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_pre_auth            = false
  home_directory_required              = false
  mns_logon_account                    = false
  password_never_expires               = false
  password_not_required                = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
  change_password_at_logon             = false
  password_wo                          = var.computer_password
  password_wo_version                  = 1
  enabled                              = true
}
```

The SDK creates disabled, optionally sets the initial password, then enables only when explicitly
requested. Accepted stages are never automatically rolled back or replayed after later failure.
The account GUID remains stable across supported same-domain rename and move operations.

This manages the directory object only. It never joins a host to the domain, rotates a joined
host's local machine secret or installs/configures an operating system. The operating-system
fields are directory metadata. Use owned, unjoined accounts for password acceptance fixtures.

Explicit `change_password_at_logon` is applied after creation/password setup in a separate typed
phase. A failed phase preserves the creation checkpoint and earlier accepted stages. When omitted,
the setting remains observed: password reset can change `pwdLastSet` without reapplying an old value.
Encryption names select only the low five account-policy bits; compound identity owns its own bit.
The SDK preserves unrelated encryption and account-control bits. No raw replacement mask is exposed.

`password_wo` is sensitive and write-only. Supply it through an ephemeral variable from your secret
manager. It is never placed in plan/state, private mutation intent or examples. Increase the
positive `password_wo_version` to request a reset. Changing a secret alone does not schedule a
reset. The stored version records acknowledgement, not password readback or login verification.
An uncertain password operation blocks further writes; it is never verified by retrieving a
password or implicitly trying a login.

Omitted optional fields retain directory values. Flags are changed individually; unrelated
account-control bits are never replaced by a provider-supplied raw mask. `account_expires` is
an exact nonnegative FILETIME: both zero and 9223372036854775807 explicitly mean never. No locale
date parsing is performed.

## Explicit property clearing

```terraform
resource "activedirectory_computer" "computer" {
  name             = "terraform-computer"
  sam_account_name = "terraform-computer$"
  parent_dn        = "OU=Automation,DC=example,DC=test"
  clear_properties = ["description", "location"]
}
```

Clears select optional LDAP text attributes only. Unavailable/hidden/ranged attributes do not
prove an empty value or a clear. Removing a configured property leaves it unmanaged.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Literal directory object name. Changes rename the same GUID.
- `parent_dn` (String) Explicit parent DN; changing it moves the object within its domain.
- `sam_account_name` (String) Explicit canonical computer sAMAccountName with one trailing $. No account name or container is inferred.

### Optional

> **NOTE**: [Write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments) are supported in Terraform 1.11 and later.

- `account_expires` (Number) Raw nonnegative FILETIME. Zero and 9223372036854775807 mean never. Omission retains the directory value.
- `account_not_delegated` (Boolean) Explicit account_not_delegated account setting; omitted values retain their observations and are never implicitly enabled.
- `allow_reversible_password_encryption` (Boolean) Explicit allow_reversible_password_encryption account setting; omitted values retain their observations and are never implicitly enabled.
- `change_password_at_logon` (Boolean) Explicit pwdLastSet semantics. On creation, applied after password setup in a separate typed phase. When omitted, observation can change after password reset without acquiring ownership.
- `clear_properties` (Set of String) Explicit optional LDAP text properties to clear on an existing object. Excludes account identity, passwords, expiration and raw flag masks.
- `compound_identity_supported` (Boolean) Owns the account's compound identity bit independently of encryption selection and unrelated flags.
- `description` (String) Typed description directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `display_name` (String) Typed displayName directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `dns_host_name` (String) Typed dNSHostName directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `does_not_require_pre_auth` (Boolean) Explicit does_not_require_pre_auth account setting; omitted values retain their observations and are never implicitly enabled.
- `enabled` (Boolean) Explicit enabled account setting; omitted values retain their observations and are never implicitly enabled.
- `home_directory_required` (Boolean) Explicit home_directory_required account setting; omitted values retain their observations and are never implicitly enabled.
- `home_page` (String) Typed wWWHomePage directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `kerberos_encryption_types` (Set of String) Explicit des_crc, des_md5, rc4, aes128 or aes256 account policy bits. Omission retains observed values; an empty set selects zero low bits. Compound identity and unrelated bits remain independent.
- `location` (String) Typed location directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `managed_by_dn` (String) Typed managedBy directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `mns_logon_account` (Boolean) Explicit mns_logon_account account setting; omitted values retain their observations and are never implicitly enabled.
- `operating_system` (String) Typed operatingSystem directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `operating_system_hotfix` (String) Typed operatingSystemHotfix directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `operating_system_service_pack` (String) Typed operatingSystemServicePack directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `operating_system_version` (String) Typed operatingSystemVersion directory property. Omission retains its observed value. Metadata does not configure the corresponding host.
- `password_never_expires` (Boolean) Explicit password_never_expires account setting; omitted values retain their observations and are never implicitly enabled.
- `password_not_required` (Boolean) Explicit password_not_required account setting; omitted values retain their observations and are never implicitly enabled.
- `password_wo` (String, Sensitive, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) Write-only UTF-8 account password supplied through an ephemeral variable. Requires an explicit positive version and never appears in plan/state. This does not join a host to the domain.
- `password_wo_version` (Number) Acknowledged password intent version. Increase with password_wo to request a reset; decreasing is refused. Import starts at zero.
- `smartcard_logon_required` (Boolean) Explicit smartcard_logon_required account setting; omitted values retain their observations and are never implicitly enabled.
- `trusted_for_delegation` (Boolean) Explicit trusted_for_delegation account setting; omitted values retain their observations and are never implicitly enabled.
- `trusted_to_auth_for_delegation` (Boolean) Explicit trusted_to_auth_for_delegation account setting; omitted values retain their observations and are never implicitly enabled.
- `use_des_key_only` (Boolean) Explicit use_des_key_only account setting; omitted values retain their observations and are never implicitly enabled.
- `user_principal_name` (String) Typed userPrincipalName directory property. Omission retains its observed value. Metadata does not configure the corresponding host.

### Read-Only

- `distinguished_name` (String) Current directory-observed distinguished name.
- `id` (String) Stable computer object GUID used for lifecycle and import identity.
- `recovery_dn` (String) Recovery locator when creation acceptance precedes GUID observation.
- `sid` (String) Directory-assigned SID when visible.

## Import

```terraform
import {
  to = activedirectory_computer.computer
  id = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
}
```

```terraform
import {
  to = activedirectory_computer.computer
  identity = {
    id = "8b48ae8e-aea5-4c45-acef-b319f8db9cf4"
  }
}
```

<!-- schema generated by tfplugindocs -->
### Identity Schema

#### Required

- `id` (String) Stable nonzero computer object GUID.

Import observes the computer but cannot recover historical password-version metadata; its version
starts at zero. Remove password input/version configuration when adopting an account unless you
explicitly intend a new reset. Failed creation may leave Terraform's object tainted even though
the directory accepted it. Preserve the GUID/recovery DN, verify the account and each accepted
stage independently, then use explicit state/import recovery without replaying creation or an
unknown password operation. DN-only candidates are never automatically adopted.

Deletion is leaf-only. It never joins a host, deletes a subtree or deletes related objects. Protected,
nonempty, denied or unverified deletion retains state and requires explicit reconciliation.
