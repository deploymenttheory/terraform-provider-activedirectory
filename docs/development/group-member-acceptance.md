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

## Outstanding lab qualification

No actual membership lab qualification is claimed yet. Before review readiness,
run real Terraform using Windows/Linux amd64 clients against Server 2016, 2019,
2022 and 2025, with independently owned fixture roots and verified protected auth.

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
