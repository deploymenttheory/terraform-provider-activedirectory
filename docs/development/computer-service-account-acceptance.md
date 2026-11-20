# Computer service account associations

CORE-09 / #36 covers the resource, singular and plural data sources, and list
resource for one ordered computer GUID/service-account GUID relationship.
It manages `msDS-HostServiceAccount` linked values through the public SDK.
No host installation, password retrieval/reset or password-reader grant occurs.

## Local checks

Run `scripts/run-unit.ps1`, `go vet ./...`, required read-only lint,
`go generate ./...` and `go run ./internal/tools/docscheck`.
Feature-local fixtures use synthetic objects and isolated provider factories.
Actual Terraform CLI unit cases exercise create/adopt, string/identity import,
generated configuration, no-change, sole-link drift repair and destruction that
preserves endpoints and unmanaged links. Other checks cover class restrictions,
bounds, hidden values, partial discovery, cancellation and consumer stop,
uncertain accepted effects, durable recovery intent and contradictory identities.

The SDK's existing `TestServiceAccountAssociationRemoveAndUncertainOutcomes`
explicitly distinguishes acknowledged last-link deletion from a readable final
collection. The provider regression first failed when the accepted deletion was
blocked solely by the omitted attribute. The implementation accepts only the
same targeted acknowledgement: accepted state, matching identities, no partial
effects, and either no error or an error tree containing only the SDK verification cause with the
specific completed omitted-attribute range result. Mixed or unrelated errors, uncertain submissions and conflicting versions retain
recovery intent. The SDK joins even a single operation error with its context
cause; wrappers alone do not turn that pure verification outcome into uncertainty. A warning
preserves the fact that collection visibility remains indeterminate. No general
readback failure becomes successful deletion and no SDK error text is exposed.

## Actual lab cases

- `TestAccComputerServiceAccountLifecycle`: sMSA associations.
- `TestAccComputerServiceAccountGroupManaged`: gMSA associations; requires a
  normally provisioned, genuinely mature KDS root on both selected DCs.
- `TestAccComputerServiceAccountDeniedAndHidden`: ordinary-reader denied write
  and independently prepared hidden association, using sMSA endpoints.

Run from Windows and Linux amd64 against Server 2016/2019/2022/2025. The fixture
root must be independently prepared and identified by GUID, DN and permanent
marker. Bind credentials and the fixture CA enter through protected process
inputs; the observer and providers use verified LDAPS. Fixture passwords are
random, cleared after submission, and checked against plans/state. The resource
never creates the endpoint accounts; the independently bounded lab helper does.

Lifecycle owns three computers and five service accounts. Terraform manages
three links: one adopted, one added and one sole link on another computer. Native
observations prove unmanaged-link preservation, stable identities after endpoint
renames, direct-link drift repair, replacement with old endpoints retained, and
last-link deletion without deleting the computer or account.

The permission fixture owns a separate disabled computer and standalone managed
account. It records exactly one property-specific ReadProperty denial for the
ordinary actor, verifies the full ACL shape and both administrative/ordinary
views, and never changes privileged memberships or global policy. The ordinary
reader's omitted attribute cannot prove absence or a complete empty collection.

Independent process retirement must be known before directory, reader or TLS
cleanup. Unknown jobs preserve their handle and fixtures; observer expiry never
authorizes replay. Retain original failed receipts, source/build hashes and
follow-up cleanup proof. Stage enough free disk for the frozen test tools before
creating directory fixtures; remove only independently verified retired public
tool copies when reclaiming space.

## Qualification status

Unit and source checks do not qualify real directory behavior. Live matrix
qualification is pending for this implementation; record exact revisions,
client/server builds, outcomes and independent cleanup before marking the PR
ready. macOS and arm64 live acceptance is not claimed. CORE-06's remaining
Server 2016 gMSA checks also require real KDS maturity; never backdate a root key.


The first actual Windows/Server 2025 run on source `59a5150` passed initial
creation/adoption, no-change and imports, then exposed a query fixture containing
a duplicate provider/unsupported terraform block. Cleanup also exposed the SDK's
single-cause `errors.Join` wrapper, which the initial provider predicate rejected.
The focused wrapper regression failed before the correction and now requires every
leaf to be the verification cause; mixed failures remain pending. The query file
now contains only its list block, reusing the surrounding test configuration.
Original receipt `cs-25-20261120t005110182z.json` and matching manifest/coordinator
are retained. All processes and directory/reader/TLS cleanup verified, and the
original baseline was restored. This failed run is not qualification evidence.


The next run, `cs-25-20261120t005756624z.json` on `e331766`, progressed past
query and imports and retired without a post-destroy reconciliation error, then
failed the independent service-account fixture rename. The copied rename helper
appended the full service-account class to its owned prefix and `-renamed-`,
producing a 78-character CN beyond the directory bound. Fixture rename now uses a
bounded GUID-derived suffix while retaining the permanent marker and original
GUID. Its original receipt and build/coordinator remain preserved; process and
all directory/reader/TLS cleanup and original baseline verified.
