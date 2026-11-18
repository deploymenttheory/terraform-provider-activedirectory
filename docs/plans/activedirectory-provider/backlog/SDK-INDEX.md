<!-- weave-backlog:activedirectory-provider:v1:INDEX -->
Backlog ID: `INDEX` · Target: `deploymenttheory/go-sdk-activedirectory` · Source: [approval package](../README.md), provenance [research](../research.md).

## Purpose and scope

Deliver portable SDK protocols/representations required by the provider, retaining the SDK v1 evidence contract. This is a reviewed backlog draft, not implementation or publication evidence. The companion baseline is [provider](INDEX.md).

## Product baseline

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

## Glossary

Execution runtime is the OS/architecture running Terraform; directory target is the AD DS endpoint. An individual relationship owns one direct edge; effective membership is only an observation. Adoption does not imply infrastructure provisioning or deletion. A raised PR alone is not completion. Full terms: [glossary](../glossary.md).

## Agreed architecture decisions

Use the [agreed architecture](../architecture.md). Portable direct SDK transports replace the Windows-only prerequisite; host-local/cloning operations are excluded. No invented provider ADR IDs apply. The current SDK v1 gate remains unmet and is not a global provider-development blocker.

## Delivery map

| Gate | Exit conditions |
| --- | --- |
| M1 — Provider foundation | Root Framework module, isolated registries/mocks, schema and docs tooling, mandatory CI and reproducible build commands exist. No live feature qualification is claimed. |
| M2 — Portable authenticated directory foundation | Portable LDAP, verified TLS, Kerberos CBT, SASL signing/sealing, outcome contracts, client integration and Linux/Windows lab connectivity pass their acceptance gates. |
| M3 — Four-surface reference slice | OU and user resource/data/list lifecycles and the unlock action have unit, docs/examples and applicable eight-cell live evidence; no-change and failure recovery are demonstrated. |
| M4 — Core directory management | Core objects, supported managed service accounts, account settings and individual relationships meet the same delivery bar. |
| M5 — Policies and access control | Password, authentication, central access, claims and resource property families pass descriptor/condition, privilege and relationship qualification. |
| M6 — Topology and observations | Topology, adopted settings, discovery and portable diagnostics cover the applicable client/server matrix without incomplete-read data loss. |
| M7 — Advanced directory actions | Explicitly acknowledged high-impact actions are qualified in isolated/restorable fixtures with outcome and recovery evidence. |
| M8 — Provider release qualification | All included surfaces, examples/templates, dependency and evidence audits, release artifacts and security checks pass; macOS live evidence remains explicitly unclaimed. |

## Requirement coverage

The [151-operation matrix](../capability-matrix.md), [type catalog](../terraform-types.md) and [preview](preview.md) account for every requirement, exclusion and implementation owner. SDK and provider tasks have separate repositories but share stable IDs and genuine native blockers.

- [EPIC-SDK](epics/EPIC-SDK.md) — [Epic SDK] Portable SDK foundation and capabilities (M7).
- [SDK-01](tasks/SDK-01.md) — [SDK-01] Expose portable backend and authentication contracts without leaking internal packages (M2).
- [SDK-02](tasks/SDK-02.md) — [SDK-02] Implement portable LDAP sessions, controls and binary-safe operations (M2).
- [SDK-03](tasks/SDK-03.md) — [SDK-03] Implement verified LDAPS and StartTLS with measured channel protection (M2).
- [SDK-04](tasks/SDK-04.md) — [SDK-04] Implement portable Kerberos authentication and TLS channel binding (M2).
- [SDK-05](tasks/SDK-05.md) — [SDK-05] Implement and qualify Kerberos SASL signing and sealing (M2).
- [SDK-06](tasks/SDK-06.md) — [SDK-06] Implement portable security descriptor parsing, editing and encoding (M2).
- [SDK-07](tasks/SDK-07.md) — [SDK-07] Implement portable authenticated replication diagnostics (M6).
- [SDK-08](tasks/SDK-08.md) — [SDK-08] Implement portable conditional expressions and descriptor conversion (M5).
- [SDK-09](tasks/SDK-09.md) — [SDK-09] Preserve portable write outcomes, normalization and reconciliation contracts (M2).
- [SDK-10](tasks/SDK-10.md) — [SDK-10] Complete portable explicit-domain discovery and topology reads (M6).
- [SDK-11](tasks/SDK-11.md) — [SDK-11] Qualify the portable authenticated backend for provider use (M2).
- [SDK-12](tasks/SDK-12.md) — [SDK-12] Qualify portable cross-domain move and delegation behavior (M7).

## Execution conventions

Use namespace `activedirectory-provider:v1`; type/priority/area/component label dimensions and milestone mappings are specified in [metadata](metadata.json). Existing actual feature issues remain context unless explicitly a direct blocker. Native parents and blockers are required, not just body links. Readiness is computed from unresolved blockers; no due dates or synthetic blocked labels are assigned. Epic acceptance remains independent of child closure. Preserve human edits and reconcile both open and closed identities before writes.
