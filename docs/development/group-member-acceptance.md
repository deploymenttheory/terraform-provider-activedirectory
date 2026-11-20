# Direct group membership acceptance

CORE-08 implements a resource, singular data source, relationship collection data
source and list resource. One resource owns one ordered group GUID/member GUID pair.
The public SDK controls protocol and native membership changes; the provider never
replaces the whole member attribute or infers a direct edge from primary or nested
membership.

Actual Terraform unit tests exercise new-link creation, existing-link adoption,
string and identity import, generated configuration, no-change planning, repair
following external removal, and deletion preserving unrelated links and both
objects. Each discovery surface has its own response and Terraform fixtures and
an isolated provider factory. List queries verify composite import identity and the
resource snapshot; cancellation and consumer stop retain bounded behavior.

A sole-member external removal also repairs through an explicit update: unknown
visibility is retained without false absence. SDK-boundary tests preserve uncertain effects and durable intent after failed
readback, reconcile a fresh same-pair observation without replay, reject unavailable
private storage and retain contradictory mutation identity evidence.

## Real lab cases

Both live suites are implemented; execution and qualification remain pending.
Before review readiness, run real Terraform using Windows/Linux amd64 clients
against Server 2016, 2019, 2022 and 2025, giving sixteen distinct cases. Fixtures
use independently owned roots and verified protected authentication.

- `TestAccGroupMemberLifecycle` adopts an existing user link and creates direct
  group/computer links. It covers string/identity import with generated config,
  bounded plural/list discovery, an actual empty group, primary/nested exclusion,
  external removal and rename, target replacement and no-change planning. An
  independent observer proves all endpoint objects and unmanaged nested links
  survive Terraform deletion before separately retiring owned fixture objects.
- `TestAccGroupMemberDeniedAndHidden` uses an independently prepared ordinary
  actor, ephemeral Terraform credentials and a marker-owned group with exactly
  one member-property read-deny ACE for that actor. Native PowerShell verifies
  the nonempty hidden group and records positive/negative filter observations.
  The provider must never convert its hidden link into confirmed absence or a
  complete empty collection. Real Terraform also proves refused link creation
  leaves existing native relationships and both endpoint objects unchanged.

The observer opens before fixture effects, checks GUID/root/marker/class/name
boundaries before mutation and retirement, and scans complete plans and state for
bind and fixture passwords. No privileged group membership or policy is changed.
Native permission fixtures remain owned by the outer envelope and are retired
only after the acceptance process is independently known to have stopped.

Required scenarios include direct user/group/computer links, imports, duplicate
adoption, external rename/removal, an actually empty group, one-edge deletion
preserving an unmanaged link, primary/nested exclusion, replacement, bounded data
and list discovery, ordinary-actor write refusal and hidden member attributes.
Verify positive and negated server assertions against native observations,
especially empty and inaccessible groups. Zero search rows alone never authorize
absence. If native behavior disproves that contract, fix the SDK boundary in a
separate PR rather than weakening state-retention rules.

Every run must retain exact source and binary/module revisions, actual client and
server builds, fixture identities, process retirement, independently verified
cleanup and restored VM/Linux baselines. macOS/arm64 build or unit checks are not
live qualification. Link mutations must never affect privileged groups, policy,
foreign trusts or directory objects outside the isolated owned root.

## Portable range dependency

Initial Windows/Server2025 run `gm-25-20261119t233630082z` acknowledged the first
independent fixture link but failed its ranged readback before Terraform applied
resources. The portable SDK session lacked the reader interface required by
`GroupMemberLinks`. Process retirement, directory/reader/TLS cleanup and baseline
restoration verified; the failed receipt and matching source/binary manifests are
retained unchanged.

[SDK PR219](https://github.com/deploymenttheory/go-sdk-activedirectory/pull/219)
implements that interface on the existing protected bounded read path. Its public
service wire regression fails before the patch and passes afterward for plain,
TTL and partial ranges. The provider pins immutable
`v0.2.1-0.20261119234403-7470f08ad229`; fixed-pin lab requalification is required.
No provider-side range workaround is introduced.

## Native visibility findings

Fixed-range Windows/Server2025 run `gm-25-20261119t234916368z` passes the complete
Terraform lifecycle, then catches a provider error in hidden-value inference.
The native fixture proves one real link while its ordinary actor sees no member
values: the positive LDAP filter returns zero groups and the negated filter
returns one. An unqualified negative filter therefore cannot establish absence.
Process retirement, directory/reader/TLS cleanup and original baselines verify.

Read-only native probe `gn-25-20261119t235319540z` uses explicit Kerberos signing
and sealing. Administrator Compare returns 6 for the existing value, 5 for a
missing value in the nonempty group, and 16 for an empty attribute. The ordinary
actor receives 16 for the hidden value and the empty attribute alike. Compare is
therefore not an empty-versus-hidden oracle. Both connections and all fixtures
were independently retired. This probe is not Terraform/platform qualification.

The corrected provider retains indeterminate presence as null `is_member` and
plans the configured present intent. A later explicit apply ensures only that
link using existing native permissive semantics; unresolved earlier writes still
block replay. Positive absence queries require an actually returned member value
as a visibility witness and the exact same group GUID/USN. They read one bounded
value rather than enumerate unrelated members. Collection omission reports a
warning with `complete = false`, including an actually empty group, instead of
inventing complete empty results. Native error-free atomic value-deletion
acknowledgement is operation-specific absence proof; failed/uncertain responses
retain recovery state.

Microsoft documents unreadable attributes as absent both in returned values and
filters in [MS-ADTS access checks](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-adts/8271b44d-a755-4872-a762-1ac57152099d).
The operation's native permissive behavior follows
[Remove-ADGroupMember](https://learn.microsoft.com/en-us/powershell/module/activedirectory/remove-adgroupmember?view=windowsserver2025-ps).


## Qualified runtime

All 16 actual Terraform cases passed on runtime
`ac511e66649066b721004e2c70df0daee658e5fd`, using SDK
`v0.2.1-0.20261119234403-7470f08ad229`, Go 1.27.4 and Terraform 1.17.2.
Documentation-only updates after qualification do not replace the frozen binaries.

| Server | Windows amd64 | Linux amd64 | Actual cases per client |
| --- | --- | --- | --- |
| 2016 | Pass | Pass | Lifecycle; denied and hidden |
| 2019 | Pass | Pass | Lifecycle; denied and hidden |
| 2022 | Pass | Pass | Lifecycle; denied and hidden |
| 2025 | Pass | Pass | Lifecycle; denied and hidden |

Lifecycle verifies user/group/computer direct links, adoption, string/identity
imports, generated config, bounded data/list discovery, primary/nested exclusion,
external rename/removal, explicit sole-link drift repair, endpoint replacement,
no-change plans, and one-link/last-link deletion preserving both endpoint objects
and independently owned unmanaged links. Each denied/hidden case independently
proves one native link, ordinary-reader omission, and the exact property-only ACE.
The actual empty-collection case also preserves indeterminate visibility.

The independent 16-case audit verified exact source, both binary hashes, module
and build-manifest hashes, actual client/server builds, native permission evidence,
process retirement, directory/reader/TLS cleanup, all 12 VM baseline states and
the restored Linux Server 2025 profile. Local retained receipts:

- Windows 2025: `gm-25-20261120t001529125z.json`.
- Linux 2025: `gm-25-20261120t002009826z.json`.
- Both 2016: `gm-16-20261120t003516089z.json`.
- Both 2019: `gm-19-20261120t004031671z.json`.
- Both 2022: `gm-22-20261120t004555956z.json`.
- Audit: `group-member-matrix-audit-20261120t004923417z.json`.

An earlier Server 2016 attempt exhausted the Linux disk before a terminal result
could be saved. Its original receipt remains unchanged and contributes no Linux
pass. Exact saved-job inspection independently verified no surviving worker or
owned test process. Only checksum-verified duplicate public Terraform executables
from completed runs were removed; test state and historical evidence were retained.
Follow-up retirement proof verified directory/reader/TLS cleanup and restored the
original baseline before the successful fresh run. Never count an observer timeout
or missing output as a pass, failure with no effects, or permission to replay.

macOS/arm64 live qualification and managed TTL behavior are not claimed here.
