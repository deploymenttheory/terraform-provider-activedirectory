# Service account acceptance

The feature's real Terraform unit suite covers gMSA CRUD, same-GUID rename/move,
reader/SPN changes, explicit clears, import/generated configuration and repeated no-change
plans. sMSA tests cover write-only creation password secrecy, buffer erasure, replacement
on password-version changes and replacement without carrying gMSA-only metadata. Failure
tests preserve accepted stage identity and refuse hidden/partial clear proof or write replay.

All three feature cases pass from Windows and Linux amd64 against Server 2025.
The required Server 2016/2019/2022 cases remain pending; missing fixture evidence
is not a passing or unsupported result. Keep the implementation PR in draft.
Use the guarded runners described in [testing](testing.md), selecting
`TestAccServiceAccountLifecycle`, `TestAccServiceAccountCreationForms` and
`TestAccServiceAccountPasswordFailure`. The feature-local
fixtures and embedded Terraform configurations are implemented.
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

The password-failure case requests a deliberately oversized fixture-only sMSA password.
It checks that accepted creation remains disabled with the same GUID, a later apply cannot
replay or replace that object, and explicit import/independent fixture retirement can recover.
This follows the documented [New-ADServiceAccount password behavior](https://learn.microsoft.com/en-us/powershell/module/activedirectory/new-adserviceaccount?view=windowsserver2025-ps#-accountpassword).
It changes no domain password policy and performs no password-read or login probe.
The Server 2025 native cmdlet accepted the initial weak-password sample, matching the SDK;
that sample could not prove a failure. An oversized value produced a native password-stage
rejection with a retained disabled object. Receipts `sr-25-20261119t201840469z-native-password`
and `sr-25-20261119t202009573z-native-password` record both comparisons and verified cleanup.

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

The successful Server 2025 runs used provider runtime revision
`1d1f2998312139be5ffc3660cb426f2da1e62ba5`, immutable SDK
`v0.2.1-0.20261119203218-b4ebca6eeb40`, Go 1.27.4 and Terraform 1.17.2.
Receipts `sa-25-20261119t204104321z` (Windows) and
`sa-25-20261119t204352949z` (Linux) record six passing cases, positive process
retirement, independent directory/TLS cleanup and restored VM/Linux baselines.
This includes actual rejected-password recovery/import with the SDK authoritative
DN fix. A subsequent checksum/documentation update removes only obsolete SDK
checksums; it changes no runtime source or selected dependency.

The first Server 2016 lifecycle request failed before completing creation.
Read-only preflight then observed zero KDS root keys on both DCs; native
`New-ADServiceAccount` failed for both the minimal and lifecycle-shaped requests
with code `-2146893811`, leaving no gMSA. Receipts
`service-account-preflight-2016-20261119t205752624z` and
`sn-16-20261119t205752775z` identify the missing LAB-03 infrastructure prerequisite.
The failed provider run `sa-16-20261119t205108524z` records verified process
retirement and independent subject/root/TLS cleanup. It does not qualify Server 2016.

The approved LAB-03 KDS fixture work must provision any missing single key in each
isolated forest, prove replication and allow the documented multi-DC maturity
interval before the remaining live matrix. Do not backdate these two-DC keys or
change SDK behavior to bypass the prerequisite. Continue other provider features
while the persistent lab infrastructure matures.
