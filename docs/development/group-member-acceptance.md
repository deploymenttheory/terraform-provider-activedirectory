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

SDK-boundary tests preserve uncertain effects and durable intent after failed
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
