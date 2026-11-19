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

Qualify both cases from Windows and Linux amd64 against Server 2016, 2019, 2022 and 2025.
Retain committed source/module/binary hashes, selected target/profile, case outcomes,
independent directory/TLS/process cleanup and restored VM/client baselines. Skipped/unrun
cells are not qualification. Live execution remains pending until the PR contains actual
receipts; unit/build coverage does not qualify macOS or arm64 live behavior.
