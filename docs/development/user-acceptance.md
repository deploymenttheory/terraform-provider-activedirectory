# User acceptance

The user feature keeps its SDK mocks, unit Terraform configurations and live Terraform fixtures
under `internal/services/resources/users/user/`. Singular/plural data sources and list discovery
also have isolated unit factories beside their implementations. The live resource suite registers
all four surfaces and uses a separate SDK observer to verify ownership and cleanup.

The live guard and execution commands are described in [testing](testing.md). Select
`TestAccUserLifecycle` or `TestAccUserPasswordFailure` through the acceptance runner. Supply
`AD_ACCEPTANCE_PARENT_DN` and `AD_ACCEPTANCE_PARENT_GUID` for an independently created OU named
`OU=sdk-portable-<run>,<partition>`, whose description is `sdk-portable-parent:<run>`.
The tests refuse a different root, GUID or marker. No existing users or organizational units
are adopted as test fixtures.

Supply distinct, strong fixture passwords through `TF_VAR_user_password` and
`AD_ACCEPTANCE_ROTATED_USER_PASSWORD`. These must be generated solely for the disposable lab
subject, passed through protected process environment and excluded from command arguments,
logs and receipts. Provider authentication still uses its separately supplied credentials.
The test checks complete JSON plans and Terraform state for either fixture password.
Avoid the user name and display-name tokens in generated passwords: AD complexity rules
reject them even when the password has sufficient length and character variety.

The lifecycle test exercises disabled creation/password/enable, a no-change plan, password
rotation, rename/move beneath the owned root, both GUID and resource-identity import, generated
configuration, singular/plural reads, list discovery, explicit property clearing and deletion.
The failure test deliberately submits a weak fixture password to the lab's complexity policy,
checks that accepted disabled creation is retained, refuses automatic mutation replay, then
explicitly retires the uniquely observed marker-owned user before destroying Terraform state.
The lab must reject that weak password; the test fails if it does not.

Every recorded user GUID must independently resolve to confirmed absence. The move destination
is also verified by GUID and marker before leaf deletion. The PowerShell fixture helper can
reconcile remaining users and move destinations only below the verified run root; it validates
the entire bounded tree before deleting any leaf. It refuses unowned descendants, nonempty
objects, changed identities and missing cleanup proof.

Run applicable cases on Windows and Linux amd64 against Server 2016, 2019, 2022 and 2025.
An unrun or skipped cell is not acceptance evidence. Keep source revision, binary hashes,
client/server versions, protected authentication mode, fixture identity and independent
cleanup receipts locally; summarize actual results in the feature PR.
