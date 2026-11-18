<!-- weave-backlog:activedirectory-provider:v1:SDK-03 -->
Backlog ID: `SDK-03` · Target: `deploymenttheory/go-sdk-activedirectory` · Epic: [EPIC-SDK](../epics/EPIC-SDK.md) · Product baseline: [INDEX](../SDK-INDEX.md)

## Outcome

Implement verified LDAPS and StartTLS with measured channel protection. The current provider is a template, and SDK support/qualification must be established rather than inferred from a planned capability.

## Implementation

Establish TLS before bind credentials, verify hostname and trust chain, enforce a TLS 1.2 minimum and expose observed protection. Support explicit private CA trust. Reject anonymous/empty-password simple binds, failed StartTLS and insecure simple bind; do not retry on plaintext or disable certificate validation.

Scope owner: sdk. Adjacent protocol, fixture and provider work remains owned by the linked predecessor tasks. All paths introduced here are planned paths, not existing implementation claims.

## Acceptance criteria

- [ ] Correct private-CA LDAPS/StartTLS works on Linux and Windows against every target version.
- [ ] Wrong hostname, expired/untrusted certificate and failed StartTLS prevent credential or mutation submission.
- [ ] Raised implementation PR; passing required CI; meaningful unit checks and applicable lab evidence (a purely offline deliverable must record why live tests are not applicable).


## Verification

In the SDK repo, run go test ./..., go vet ./..., required lint and portable backend-specific tests. LAB-02 supplies approved Linux/Windows live runs against Server 2016/2019/2022/2025 for this ticket's applicable protocol/security behavior. Add malformed-response, refusal, partial-result and connection-loss cases at the actual session seam. Preserve historical evidence bytes and record fresh provenance; no existing native result qualifies a new backend.

## Dependencies

Direct blockers:

- [SDK-02](SDK-02.md) — required predecessor: Implement portable LDAP sessions, controls and binary-safe operations.

## Applicable product rules

- **R1:** Use Terraform Plugin Framework, protocol V6 and Terraform >=1.14; follow the Microsoft 365 provider architecture and file patterns.
- **R2:** Provider execution is portable across Linux/Windows/macOS; AD DS is the Windows Server target. No Windows gateway/helper is part of the selected architecture.
- **R3:** First usable release includes verified TLS simple bind, portable Kerberos with TLS channel binding, and Kerberos SASL signing/sealing. Never downgrade protection or log/store credentials.
- **R4:** Live acceptance uses Linux and Windows clients against applicable Server 2016/2019/2022/2025. macOS has build/unit coverage, not claimed live acceptance; amd64 live evidence does not qualify arm64.
- **R5:** Every implementation ticket needs a raised PR, passing required CI, unit tests, applicable lab evidence, examples and template-generated docs before completion. Merges require separate authorization.
- **R6:** Object identity is GUID; names/DNs may change. Relationship identities own one edge. Authoritative collection resources are deferred; avoid competing attribute ownership.
- **R7:** Preserve accepted effects, partial observations and uncertain writes. Confirm absence before state removal; reconcile before another write. No blind mutation retry or automatic rollback.
- **R8:** Settings resources adopt existing infrastructure and relinquish management on delete. Never delete/reset a domain/forest as an ownership-removal side effect. Relationship deletion removes the owned link only.
- **R9:** High-impact/irreversible directory changes are explicit actions with operation-specific acknowledgement, capability checks and isolated/restorable acceptance fixtures.
- **R10:** The four host-local service-account operations and two DC cloning operations are excluded from every provider surface. AD LDS, ephemerals, provider functions and authoritative collections are outside this initial programme.
- **R11:** Develop and release qualified subsets behind per-feature SDK gates, not the complete SDK v1 gate. Reuse existing SDK tickets; do not claim all 151 operations implemented or qualified.
- **R12:** Run lab PowerShell directly through the approval mechanism. Never use RunAs/elevation or route around prompts. Preserve historical evidence bytes and independent cleanup checks.

Full source: [approved product rules](../../README.md#product-rules). Implementation must preserve these constraints without relying on this chat.

## Planning

Milestone: M2 — Portable authenticated directory foundation.
Priority: P0 — foundation/integrity/release blocker.
Component / scope owner: sdk; destination `deploymenttheory/go-sdk-activedirectory`.
Source rationale: [implementation sequence](../../implementation-plan.md) and [capability mapping](../../capability-matrix.md), source revisions in [research](../../research.md).
Suggested PR title: `feat(sdk): implement verified LDAPS and StartTLS with measured channel protection`.

## Architecture decision alignment

Follow the agreed [provider architecture](../../architecture.md), especially portable execution, public SDK boundaries, individual ownership and preserved outcomes. [Glossary](../../glossary.md) distinguishes execution runtime from directory target and observation from qualification. No provider ADR number is claimed; existing SDK ADRs remain applicable to SDK-owned contracts. See the linked baseline for the agreed decision summary.
