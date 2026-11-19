# Group acceptance

Feature-local live Terraform configurations are under
`internal/services/resources/groups/group/tests/terraform/acceptance/`. The suite registers
the resource, both data sources and the list resource through the production SDK constructor.
Each test also opens an independent SDK observer for fixture ownership and cleanup checks.

Follow the authorization and execution rules in [testing](testing.md). Supply the GUID and DN
of an independently created root named `OU=sdk-portable-<run>,<partition>`, whose description
is `sdk-portable-parent:<run>`. The test verifies that root before using it. Fixture groups and
move destinations use deterministic run-prefixed names, fresh GUIDs and a platform/run marker.
Only the selected verified LDAPS endpoint is used; no authentication fallback is attempted.

`TestAccGroupLifecycle` exercises creation, no-change plans, rename/move, category/scope
conversion, singular/plural reads, GUID and resource-identity import, generated configuration,
list snapshots, explicit clearing and deletion. The independent observer verifies retained GUIDs
and confirmed absence after deletion, and verifies the move destination before leaf cleanup.

`TestAccGroupMembersAndConstraints` adds a newly created universal group as a member of the
managed universal group. Membership belongs to the fixture, not the resource schema. A
no-change plan proves that it does not acquire membership ownership. An attempted global-scope
conversion must be rejected by the server; import must retain the prior universal scope and
GUID. After deleting the managed group, the separate member must still exist with its original
GUID, marker and universal scope. Only then does fixture cleanup delete that separate group.
No existing group, account or membership is modified.

The SDK owns atomic exact-old-value groupType updates and preservation of unrelated bits.
Provider unit tests enforce semantic requests and cover rejected concurrent changes, verified
SDK no-ops, accepted rename before rejected scope changes, tainted creation recovery and
uncertain mutation replay refusal. These synthetic cases do not claim live concurrency or
the ability to install arbitrary reserved groupType bits on a domain controller.

Run applicable cases on Windows/Linux amd64 against Server 2016/2019/2022/2025. Each completed
cell requires source/binary revisions, tool/client/server versions, authentication mode, owned
fixture identity and independently verified directory/TLS cleanup. Restore the original VM
states and Linux profile. Missing or skipped cells are not qualification; macOS/arm64 builds
and unit checks are separate from live evidence. Retain bulk logs locally and summarize actual
results in the implementation PR.
