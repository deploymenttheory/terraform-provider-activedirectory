# Service account acceptance

The feature's real Terraform unit suite covers gMSA CRUD, same-GUID rename/move,
reader/SPN changes, explicit clears, import/generated configuration and repeated no-change
plans. sMSA tests cover write-only creation password secrecy, buffer erasure, replacement
on password-version changes and replacement without carrying gMSA-only metadata. Failure
tests preserve accepted stage identity and refuse hidden/partial clear proof or write replay.

Live provider qualification remains pending. The required clients are Windows/Linux amd64
against Server 2016, 2019, 2022 and 2025; missing fixture evidence is not a passing result.
Use the guarded runners described in [testing](testing.md), selecting
`TestAccServiceAccountLifecycle`, `TestAccServiceAccountCreationForms` and
`TestAccServiceAccountPasswordFailure`. The feature-local
fixtures and embedded Terraform configurations now exist; actual results remain pending.
Never retrieve a managed password to verify a resource.

The lifecycle case creates an ordinary gMSA, changes typed settings, SPNs and selected reader
SIDs, renames/moves the same GUID, exercises both lookup forms, import/generated configuration
and actual list snapshots, clears selected values and repairs independently introduced drift.
The separate observer seeds one unrelated WP-only ACE and checks its exact bytes and the
descriptor owner/group/SACL components after reader replacement and clearing.

The creation-form case covers SDK-generated sMSA passwords, an ephemeral explicit sMSA
creation password and version replacement, and outbound-only gMSA creation. Three live
objects exercise page-size-one discovery, complete/bounded results and list consumer limits.
Plans/state are checked for bind and creation secrets. Every subject GUID is tracked through
independent absence checks; auxiliary groups/OUs retain permanent markers and leaf-only cleanup.

The password-failure case requests a deliberately noncompliant fixture-only sMSA password.
It checks that accepted creation remains disabled with the same GUID, a later apply cannot
replay or replace that object, and explicit import/independent fixture retirement can recover.
This follows the documented [New-ADServiceAccount password behavior](https://learn.microsoft.com/en-us/powershell/module/activedirectory/new-adserviceaccount?view=windowsserver2025-ps#-accountpassword).
It changes no domain password policy and performs no password-read or login probe.

`tests/fixtures/prerequisites.ps1` beside the resource provides a read-only preflight helper.
It verifies the selected AD DS target and reads explicit service-account class definitions
and KDS GUID/time metadata. It requests no KDS key data or managed-password attributes and
does not create keys, raise functional levels or mutate directory objects.

On 19 November 2026 both owned Server 2025 lab DCs reported sMSA/gMSA classes, DC functionality
10 and one shared mature eligible KDS key. Receipt
`service-account-preflight-2025-20261119t195300617z` records that observation. This is prerequisite
evidence only; it does not qualify creation, reader-policy behavior, authorization or other
server versions. Live cases must additionally verify actual effects, unrelated ACE preservation,
process/directory/TLS cleanup, source/binary provenance and restored lab baselines.
