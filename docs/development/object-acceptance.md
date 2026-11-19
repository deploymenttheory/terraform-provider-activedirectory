# Generic object acceptance

The guarded suite lives beside `internal/services/resources/objects/object/`. It registers
all four actual production object surfaces and opens a separately authenticated SDK observer.
Use the commands in [testing](testing.md), selecting `TestAccObjectLifecycle` and
`TestAccObjectNonemptyLeaf`. Provider authentication uses verified LDAPS; credentials stay
in protected process environment and are checked against retained Terraform plans/state.

Supply `AD_ACCEPTANCE_PARENT_DN` and `AD_ACCEPTANCE_PARENT_GUID` for an independently created
OU named `OU=sdk-portable-<run>,<partition>` with description `sdk-portable-parent:<run>`.
The observer verifies that GUID, DN and marker before fixture effects. Every subject and
auxiliary has a bounded run/test/platform name and permanent description marker. The
attribute being cleared is displayName, never the ownership marker.

The lifecycle case covers contact creation, binary thumbnailPhoto replacement, selective value
add/remove while preserving an independently added value, same-GUID rename/move, normalized
manager DN readback, selected-attribute JSON import, identity import/generated configuration,
GUID/DN lookup, actual paged and caller-bounded queries, list snapshots, explicit clearing,
independent drift and no-change plans. Its manager contact and move OU are separately owned
and retired through positive same-GUID leaf-deletion proof.

The nonempty case uses the generic schema-selected organizationalUnit class, checking its
OU naming attribute. It creates a separately owned contact below that subject, expects
leaf-only deletion to fail, independently retires the child and then verifies normal
management/destroy. No recursive deletion or implicit protection change is allowed.

Every captured subject GUID must independently resolve to confirmed absence. Auxiliary
cleanup verifies original GUID/name/class/DN/marker and refuses descendants. The outer
PowerShell helper validates the entire bounded tree before retiring only marker-owned
contacts/OUs beneath the original root. Unknown classes, moved objects, changed ownership,
nonempty leaves or uncertain process retirement stop cleanup and preserve evidence.

Both cases passed with Terraform 1.17.2 and Go 1.27.4 on 19 November 2026:

| AD DS target | Windows amd64 | Linux amd64 |
| --- | --- | --- |
| Server 2016 | 2/2 passed | 2/2 passed |
| Server 2019 | 2/2 passed | 2/2 passed |
| Server 2022 | 2/2 passed | 2/2 passed |
| Server 2025 | 2/2 passed | 2/2 passed |

All 16 cases used source `d01bf14b37bcfc8fdd34e4694de2bb8fac88b18a` with identical
module/build-manifest and Windows/Linux binary hashes throughout the matrix. The successful
run IDs are `ob-25-20261119t184506031z`, `ob-25-20261119t185919690z`,
`ob-16-20261119t190228700z`, `ob-19-20261119t190734647z` and
`ob-22-20261119t191221574z`. Process, directory and TLS cleanup was independently verified
for each run. The final matrix audit checks all 16 distinct case keys, profile identity,
build provenance and restoration of the original VM power states and Linux client profile.
Raw receipts remain local rather than adding generated test logs to the repository.

The first contact fixture failed because its selected class does not permit jpegPhoto.
Schema inspection identified thumbnailPhoto as the supported binary attribute, and the
fixtures/examples were corrected without relaxing the binary assertions. An earlier Linux
worker process-inspection failure preserved its fixtures; independent retirement/cleanup
was verified before a corrected run. Those earlier runs are excluded from the passing matrix.

Retain the same provenance, independent cleanup and baseline checks for future runs.
Skipped/unrun cells are not qualification. macOS has unit/build coverage and arm64 has
cross-build coverage; live behavior on those clients remains unqualified.
