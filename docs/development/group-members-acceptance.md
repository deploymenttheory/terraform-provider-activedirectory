# Group members data source

CORE-10 / #37 implements `activedirectory_group_members`, a read-only direct or
explicitly recursive query. It owns no objects or links. The provider follows the
Microsoft 365 repository's feature-local model, construct/read/state, isolated
factory/mock, examples and template/generated-documentation layout.

## Public SDK boundary

The qualified provider session exposes `GroupMembershipQueries`: ResolveIdentity
and GroupMembers only. Explicit finite object/group/range/depth/byte budgets are
validated before I/O. The selected provider connection is the authenticated
context; the provider does not open inferred foreign endpoints or forward credentials.
The SDK's required portable member-range support is pinned immutably as
`v0.2.1-0.20261119234403-7470f08ad229` from SDK PR219.

Root GUID, class, explicitly selected DN and positive candidates must agree.
Traversal GUID/version anchors are validated without requiring a DN attribute
that the ranged operation does not select. Principal GUID/SID attributes must
agree with typed identities; FSP references never become resolved account IDs.
Conflicting duplicate identity/name observations are omitted with diagnostics;
agreeing paths merge primary flags and sort deterministically. Known range and
resolution errors retain usable rows with incomplete diagnostics. A mixed limit
and unrelated failure remains an error rather than a harmless budget warning.

An empty or hidden member attribute cannot prove an empty graph. Every visited
group needs returned direct values before visibility is known. Primary-only
principals remain usable while `visibility_known=false` and `complete=false`.
Completion describes requested observations, not effective access, visibility of
every directory object, or a transactionally consistent graph. Foreign/cross-
context references unresolved in this connection retain bounded separate records
and diagnostics. No extra credentialed-context configuration or live foreign-
forest qualification is claimed by these single-context examples.

## PowerShell behavior checked

Microsoft documents recursive leaf behavior in
[Get-ADGroupMember](https://learn.microsoft.com/en-us/powershell/module/activedirectory/get-adgroupmember?view=windowsserver2025-ps).
A separate read-only native Server 2025 comparison verified that the cmdlet also
includes primary members even when no `member` values exist:

| Domain group RID | Direct member attribute values | Primary matches | Cmdlet members | Primary overlap |
| --- | --- | --- | --- | --- |
| 513 | 0 | 2 | 2 | 2 |
| 515 | 0 | 1 | 1 | 1 |
| 516 | 0 | 2 | 2 | 2 |

Native local receipt: `core10-native-primary-observation.json`. No built-in group
membership, principal, policy or ACL was changed. The SDK already includes these
primary relationships; no SDK patch or provider filtering workaround was needed.
These native observations are not Terraform or full-matrix qualification.

## Tests and live gate

Unit cases use actual Terraform CLI refreshes over explicit synthetic SDK responses:
direct/recursive results, agreeing duplicate paths, cycle-safe traversals, primary-
only results, hidden omission, bounds, scope rejection before traversal and mixed
private failures. Projection tests retain known partial rows and unresolved
references, refuse wrong anchors and conflicting duplicates, and distinguish
omitted from complete-empty results. Client tests prove the existing qualified
session and lifetime are shared without another dial.

Committed real-lab suites:

- `TestAccGroupMembersDiscovery`: owned cyclic/nested graph with duplicate paths,
  user/group/computer results, all five aggregate budgets, actual empty group,
  repeated no-change and external endpoint rename. Independently verify the graph
  and all objects remain intact after read-only Terraform operations.
- `TestAccGroupMembersPrimary`: an owned disabled user's unchanged primary group,
  read only; verify the user's GUID is returned with primary=true while direct-
  attribute visibility remains indeterminate. The built-in group is never owned
  or modified by cleanup.
- `TestAccGroupMembersVisibility`: separately prepared ordinary actor and native
  nonempty group with one exact property-specific read denial. Verify direct and
  recursive queries retain incomplete empty observations rather than false absence.

Run applicable suites from Windows/Linux amd64 against Server 2016/2019/2022/2025.
Fixtures require independently identified root GUID/DN/marker and verified LDAPS;
secrets enter protected process inputs, not committed examples. SDK observer and
provider sessions qualify separately. All markers/classes/GUIDs are rechecked
before effects and leaf cleanup. State/plans are scanned for bind and fixture
passwords; no password material or raw peer text is projected.

Preserve original failed receipts/build/coordinator hashes. Unknown process
retirement retains handles and fixtures; an observer deadline never authorizes a
restart. Independently verify process, directory/reader/TLS cleanup and original
VM/Linux baselines before qualification. Missing foreign-context fixtures or
synthetic responses do not qualify live foreign-forest variants. Live matrix
qualification is pending. macOS/arm64 live acceptance is not claimed.
