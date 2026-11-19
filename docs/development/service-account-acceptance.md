# Service account acceptance

The feature's real Terraform unit suite covers gMSA CRUD, same-GUID rename/move,
reader/SPN changes, explicit clears, import/generated configuration and repeated no-change
plans. sMSA tests cover write-only creation password secrecy, buffer erasure, replacement
on password-version changes and replacement without carrying gMSA-only metadata. Failure
tests preserve accepted stage identity and refuse hidden/partial clear proof or write replay.

Live provider qualification remains pending. The required clients are Windows/Linux amd64
against Server 2016, 2019, 2022 and 2025; missing fixture evidence is not a passing result.
Use the guarded runners described in [testing](testing.md) after the feature's live fixtures
and acceptance cases are available. Never retrieve a managed password to verify a resource.

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
