---
page_title: "activedirectory_account_settings Resource - Active Directory"
subcategory: "Accounts"
description: |-
  Adopt an existing account and manage explicitly configured account-control and expiration settings.
---

# activedirectory_account_settings (Resource)

Adopt an existing account and manage only explicitly configured account-control/expiration settings. Destroy relinquishes ownership without deleting or resetting the account. Do not co-own the same fields in typed account resources or other Terraform states; cross-workspace conflicts are not detected.

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

## Adopt an existing account

```terraform
# Adopt an existing account. This resource does not create it.
# Do not also configure this account's enabled setting in a typed account resource.
resource "activedirectory_account_settings" "example" {
  account_guid = "b591fb30-6292-4744-bf23-5f346cb74559"
  enabled      = false
}
```

The account must already exist. Create adopts its GUID and applies configured settings;
no user, computer or managed service account is created. DN/name changes made elsewhere
retain the same GUID. Changing `account_guid` replaces management: the previous account
is retained, then the new existing account is adopted.

Destroy relinquishes ownership without deleting the account or restoring former settings.
Removing an optional setting from configuration leaves that field unmanaged. Do not co-own
the same fields in a typed user/computer/service-account resource or another Terraform state.
The provider does not detect conflicting ownership across workspaces.

## Explicit account settings and finite expiration

```terraform
# Separate recipe: manage this existing account only through this settings resource
# for the fields below. Omitted fields remain unmanaged.
resource "activedirectory_account_settings" "maximal" {
  account_guid                         = "6d682f3f-6a4d-4beb-ab06-6d6115b256e1"
  enabled                              = true
  account_not_delegated                = true
  allow_reversible_password_encryption = false
  does_not_require_preauth             = false
  home_directory_required              = false
  mns_logon_account                    = false
  password_never_expires               = false
  password_not_required                = false
  smartcard_logon_required             = false
  trusted_for_delegation               = false
  trusted_to_auth_for_delegation       = false
  use_des_key_only                     = false
  change_password_at_logon             = true
  expires_at                           = "2031-04-05T11:34:56Z"
}
```

Only configured flags enter the SDK's typed atomic patch; unrelated stored bits are preserved.
`locked_out` and `password_expired` are selected-server computed observations and are never
written. Unlock is a separate explicit action. No raw account-control replacement mask or
password setter is exposed here.

`change_password_at_logon` controls `pwdLastSet`, not a stored UAC bit. True requires a password
change at next logon; false requests the server's current timestamp. An already-matching
setting is not replayed. True conflicts with `password_never_expires = true`.

`expires_at` uses canonical UTC RFC3339 with at most directory-supported 100ns precision;
locale dates, relative times and unresolved values are refused before any mutation phase.
`never_expires = false` requires an explicit finite `expires_at`. Expiration input is fully
validated before an earlier configured flag change can be submitted.

## Explicitly clear expiration

```terraform
# Separate recipe: explicitly clear expiration on an existing account.
# Both native never-expire representations read as never_expires = true.
resource "activedirectory_account_settings" "never" {
  account_guid  = "6d682f3f-6a4d-4beb-ab06-6d6115b256e1"
  never_expires = true
}
```

Omitting expiration settings performs no clear. Explicit `never_expires = true` uses the native
SDK clear operation. The computed `account_expires` preserves both native never-expire
representations, zero and 9223372036854775807; neither becomes a fabricated finite date.

## Recovery and partial visibility

Accepted or uncertain writes retain the adopted GUID, known effects and nonsecret private
intent. A fresh complete same-GUID lookup must positively match the submitted phase before
that checkpoint is cleared. Hidden attributes or historical state cannot prove a change.
A contradictory returned GUID is retained for independent reconciliation.

An unresolved checkpoint blocks later writes and automatic tainted replacement, including
relinquishment that could otherwise discard uncertainty and submit again. There is no blind
mutation retry or rollback. Independently verify the account and prior effects, then use
explicit GUID import to recover management. Import does not reset settings.

Unavailable optional observations retain prior known resource values with a warning; this is
not current verification. Data sources report unavailable values as null. State is removed
only for pure confirmed absence of the same GUID, not for permission or transport failures.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `account_guid` (String) Explicit existing account object GUID. Changing the target relinquishes the previous account and adopts the new one; neither account is created or deleted.

### Optional

- `account_not_delegated` (Boolean) Explicit typed account_not_delegated account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `allow_reversible_password_encryption` (Boolean) Explicit typed allow_reversible_password_encryption account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `change_password_at_logon` (Boolean) Explicit pwdLastSet intent: true requires change at next logon; false asks the server to set its current password timestamp. Already-matching intent is not replayed. Omission leaves it unmanaged.
- `does_not_require_preauth` (Boolean) Explicit typed does_not_require_preauth account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `enabled` (Boolean) Explicit typed enabled account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `expires_at` (String) Finite canonical UTC RFC3339 expiration at 100ns precision or coarser. Omission owns no expiration. Set never_expires = true to clear explicitly.
- `home_directory_required` (Boolean) Explicit typed home_directory_required account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `mns_logon_account` (Boolean) Explicit typed mns_logon_account account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `never_expires` (Boolean) Explicit true clears expiration through the SDK. False requires expires_at. Omission owns no expiration; both native never-expire representations are observed as true.
- `password_never_expires` (Boolean) Explicit typed password_never_expires account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `password_not_required` (Boolean) Explicit typed password_not_required account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `smartcard_logon_required` (Boolean) Explicit typed smartcard_logon_required account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `trusted_for_delegation` (Boolean) Explicit typed trusted_for_delegation account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `trusted_to_auth_for_delegation` (Boolean) Explicit typed trusted_to_auth_for_delegation account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.
- `use_des_key_only` (Boolean) Explicit typed use_des_key_only account-control setting. Omission leaves the setting unmanaged; a visible false remains distinct from an unavailable value.

### Read-Only

- `account_expires` (Number) Observed raw nonnegative FILETIME expiration. Zero and 9223372036854775807 are distinct native never-expire representations; this output is never written directly.
- `distinguished_name` (String) Current observed distinguished name; external rename/move retains GUID identity.
- `id` (String) Stable existing account object GUID and import identity.
- `locked_out` (Boolean) Selected DC's computed lockout state when visible. This resource never writes a computed flag or unlocks an account.
- `password_expired` (Boolean) Selected DC's computed password-expiry state when visible. This value is never written.
- `password_last_set` (Number) Observed pwdLastSet FILETIME when visible; unavailable is null.

## Import

```terraform
import {
  to = activedirectory_account_settings.example
  id = "b591fb30-6292-4744-bf23-5f346cb74559"
}
```

```terraform
import {
  to = activedirectory_account_settings.example
  identity = {
    id = "b591fb30-6292-4744-bf23-5f346cb74559"
  }
}
```

<!-- schema generated by tfplugindocs -->
### Identity Schema

#### Required

- `id` (String) Stable existing account GUID.
