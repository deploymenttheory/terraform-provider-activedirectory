# Computer acceptance

The computer feature keeps SDK mocks, Terraform unit configurations and live fixtures beside
`internal/services/resources/computers/computer/`. The live suite registers the resource,
singular/plural data sources and list resource. An independently authenticated SDK observer
verifies the external parent by GUID and marker before any subject is created.

Use the guarded commands in [testing](testing.md). Select `TestAccComputerLifecycle` and
`TestAccComputerPasswordObservation`. Supply `AD_ACCEPTANCE_PARENT_DN` and
`AD_ACCEPTANCE_PARENT_GUID` for an independently created OU named
`OU=sdk-portable-<run>,<partition>` with description `sdk-portable-parent:<run>`.
All subjects must be newly created, marker-owned, unjoined computer accounts. Existing host
accounts are never adopted, joined, configured or used for password operations.

Supply distinct strong disposable secrets through `TF_VAR_computer_password` and
`AD_ACCEPTANCE_ROTATED_COMPUTER_PASSWORD`. Keep them in protected process environment, out of
arguments, logs and receipts. Provider authentication uses separate credentials with verified
LDAPS. The tests inspect complete JSON plans and retained Terraform attributes for fixture secrets.

The lifecycle case covers disabled creation/password/explicit enablement, repeated no-change
plans, same-GUID rename/move, password rotation with explicit logon-password intent, named
encryption selection and independent compound identity changes, exact expiration sentinels,
GUID and identity import, generated configuration, both data sources, list discovery, explicit
text clearing and leaf deletion. Its move destination is independently verified and retired.

The password-observation case starts with a disabled account without a password. It verifies
that password reset changes observed `pwdLastSet` when `change_password_at_logon` is omitted,
then verifies that an explicitly configured setting is reapplied after a later reset without
enabling the account. It checks a no-change plan after every stage. Computer password policy
is not assumed to have user-account complexity behavior; rejection/uncertainty contracts are
exercised through isolated SDK mocks.

All recorded subject GUIDs must independently resolve to confirmed absence. The PowerShell
fixture helper validates the entire bounded tree before reconciling remaining marker-owned
computer leaves and move destinations. Unknown classes, changed GUID/markers, moved objects,
nonempty leaves and missing cleanup proof stop cleanup. Protection is never disabled implicitly.

Qualify both cases from Windows and Linux amd64 against Server 2016, 2019, 2022 and 2025.
Unrun/skipped cells do not count. Retain source revision, both binary hashes, tool versions,
target/profile identity and independent cleanup receipts locally. Report actual completed
coverage in the PR; build/unit checks do not imply macOS or arm64 live qualification.
