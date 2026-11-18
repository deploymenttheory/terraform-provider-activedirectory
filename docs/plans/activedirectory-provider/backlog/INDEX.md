<!-- weave-backlog:activedirectory-provider:v1:INDEX -->
Backlog ID: `INDEX` · Target: `deploymenttheory/terraform-provider-activedirectory` · Source: [approval package](../README.md), provenance [research](../research.md).

## Purpose and scope

Deliver an AD DS Terraform Plugin Framework provider with resources, data sources, list resources and actions, portable execution and per-feature lab qualification. This is a reviewed backlog draft, not implementation or publication evidence. The companion baseline is [SDK](SDK-INDEX.md).

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

- [EPIC-FND](epics/EPIC-FND.md) — [Epic FND] Provider foundation (M2).
- [EPIC-LAB](epics/EPIC-LAB.md) — [Epic LAB] Acceptance infrastructure (M4).
- [EPIC-CORE](epics/EPIC-CORE.md) — [Epic CORE] Core directory management (M7).
- [EPIC-POL](epics/EPIC-POL.md) — [Epic POL] Policies and access control (M5).
- [EPIC-TOPO](epics/EPIC-TOPO.md) — [Epic TOPO] Replication topology (M6).
- [EPIC-READ](epics/EPIC-READ.md) — [Epic READ] Discovery and observations (M7).
- [EPIC-DIR](epics/EPIC-DIR.md) — [Epic DIR] Adopted directory settings (M6).
- [EPIC-ACT](epics/EPIC-ACT.md) — [Epic ACT] Explicit directory actions (M7).
- [EPIC-REL](epics/EPIC-REL.md) — [Epic REL] Integrated release qualification (M8).
- [FND-01](tasks/FND-01.md) — [FND-01] Create the root Terraform Plugin Framework provider module (M1).
- [FND-02](tasks/FND-02.md) — [FND-02] Add explicit production registrations and isolated provider factories (M1).
- [FND-03](tasks/FND-03.md) — [FND-03] Integrate portable SDK clients, credentials and session lifetime (M2).
- [FND-04](tasks/FND-04.md) — [FND-04] Build SDK-boundary mocks and Terraform unit/acceptance harnesses (M2).
- [FND-05](tasks/FND-05.md) — [FND-05] Add example and template-based documentation generation (M1).
- [FND-06](tasks/FND-06.md) — [FND-06] Establish required CI and reproducible multi-platform builds (M1).
- [FND-07](tasks/FND-07.md) — [FND-07] Implement stable identities, lifecycle diagnostics and bounded readback helpers (M2).
- [LAB-01](tasks/LAB-01.md) — [LAB-01] Inventory versioned lab targets and acceptance fixture requirements (M1).
- [LAB-02](tasks/LAB-02.md) — [LAB-02] Provision safe Linux and Windows acceptance connectivity (M2).
- [LAB-03](tasks/LAB-03.md) — [LAB-03] Provision feature-specific advanced and destructive lab fixtures (M4).
- [CORE-01](tasks/CORE-01.md) — [CORE-01] Implement organizational unit lifecycle and discovery (M3).
- [CORE-02](tasks/CORE-02.md) — [CORE-02] Implement user lifecycle and discovery (M3).
- [CORE-03](tasks/CORE-03.md) — [CORE-03] Implement group lifecycle and discovery (M4).
- [CORE-04](tasks/CORE-04.md) — [CORE-04] Implement computer lifecycle and discovery (M4).
- [CORE-05](tasks/CORE-05.md) — [CORE-05] Implement object lifecycle and discovery (M4).
- [CORE-06](tasks/CORE-06.md) — [CORE-06] Implement service account lifecycle and discovery (M4).
- [CORE-07](tasks/CORE-07.md) — [CORE-07] Implement account settings adopted (M4).
- [CORE-08](tasks/CORE-08.md) — [CORE-08] Implement group member lifecycle and discovery (M4).
- [CORE-09](tasks/CORE-09.md) — [CORE-09] Implement computer service account lifecycle and discovery (M4).
- [CORE-10](tasks/CORE-10.md) — [CORE-10] Implement group members data (M4).
- [CORE-11](tasks/CORE-11.md) — [CORE-11] Implement principal group membership data (M4).
- [CORE-12](tasks/CORE-12.md) — [CORE-12] Implement account authorization groups data (M4).
- [CORE-13](tasks/CORE-13.md) — [CORE-13] Implement account search data (M4).
- [POL-01](tasks/POL-01.md) — [POL-01] Implement fine grained password policy lifecycle and discovery (M5).
- [POL-02](tasks/POL-02.md) — [POL-02] Implement fine grained password policy subject lifecycle and discovery (M5).
- [POL-03](tasks/POL-03.md) — [POL-03] Implement domain password policy adopted (M5).
- [POL-04](tasks/POL-04.md) — [POL-04] Implement user resultant password policy data (M5).
- [POL-05](tasks/POL-05.md) — [POL-05] Implement authentication policy lifecycle and discovery (M5).
- [POL-06](tasks/POL-06.md) — [POL-06] Implement authentication policy silo lifecycle and discovery (M5).
- [POL-07](tasks/POL-07.md) — [POL-07] Implement authentication policy silo access lifecycle and discovery (M5).
- [POL-08](tasks/POL-08.md) — [POL-08] Implement account authentication assignment adopted (M5).
- [POL-09](tasks/POL-09.md) — [POL-09] Implement central access rule lifecycle and discovery (M5).
- [POL-10](tasks/POL-10.md) — [POL-10] Implement central access policy lifecycle and discovery (M5).
- [POL-11](tasks/POL-11.md) — [POL-11] Implement central access policy member lifecycle and discovery (M5).
- [POL-12](tasks/POL-12.md) — [POL-12] Implement claim type lifecycle and discovery (M5).
- [POL-13](tasks/POL-13.md) — [POL-13] Implement claim transform policy lifecycle and discovery (M5).
- [POL-14](tasks/POL-14.md) — [POL-14] Implement claim transform link lifecycle and discovery (M5).
- [POL-15](tasks/POL-15.md) — [POL-15] Implement resource property lifecycle and discovery (M5).
- [POL-16](tasks/POL-16.md) — [POL-16] Implement resource property list lifecycle and discovery (M5).
- [POL-17](tasks/POL-17.md) — [POL-17] Implement resource property list member lifecycle and discovery (M5).
- [POL-18](tasks/POL-18.md) — [POL-18] Implement resource property value types data (M5).
- [POL-19](tasks/POL-19.md) — [POL-19] Implement authentication policy expression data (M5).
- [TOPO-01](tasks/TOPO-01.md) — [TOPO-01] Implement replication site lifecycle and discovery (M6).
- [TOPO-02](tasks/TOPO-02.md) — [TOPO-02] Implement replication site link lifecycle and discovery (M6).
- [TOPO-03](tasks/TOPO-03.md) — [TOPO-03] Implement replication site link bridge lifecycle and discovery (M6).
- [TOPO-04](tasks/TOPO-04.md) — [TOPO-04] Implement replication subnet lifecycle and discovery (M6).
- [TOPO-05](tasks/TOPO-05.md) — [TOPO-05] Implement replication connection settings adopted (M6).
- [TOPO-06](tasks/TOPO-06.md) — [TOPO-06] Implement rodc password replication policy member lifecycle and discovery (M6).
- [READ-01](tasks/READ-01.md) — [READ-01] Implement domain data (M6).
- [READ-02](tasks/READ-02.md) — [READ-02] Implement forest data (M6).
- [READ-03](tasks/READ-03.md) — [READ-03] Implement domain controller data (M6).
- [READ-04](tasks/READ-04.md) — [READ-04] Implement trust data (M6).
- [READ-05](tasks/READ-05.md) — [READ-05] Implement root dse data (M6).
- [READ-06](tasks/READ-06.md) — [READ-06] Implement optional features data (M6).
- [READ-07](tasks/READ-07.md) — [READ-07] Implement deleted objects data (M7).
- [READ-08](tasks/READ-08.md) — [READ-08] Implement rodc password replication policy data (M6).
- [READ-09](tasks/READ-09.md) — [READ-09] Implement account resultant password replication policy data (M6).
- [READ-10](tasks/READ-10.md) — [READ-10] Implement rodc password replication policy usage data (M6).
- [READ-11](tasks/READ-11.md) — [READ-11] Implement replication failures data (M6).
- [READ-12](tasks/READ-12.md) — [READ-12] Implement replication partner metadata data (M6).
- [READ-13](tasks/READ-13.md) — [READ-13] Implement replication queue data (M6).
- [READ-14](tasks/READ-14.md) — [READ-14] Implement replication attribute metadata data (M6).
- [READ-15](tasks/READ-15.md) — [READ-15] Implement replication up to dateness vectors data (M6).
- [DIR-01](tasks/DIR-01.md) — [DIR-01] Implement domain settings adopted (M6).
- [DIR-02](tasks/DIR-02.md) — [DIR-02] Implement forest settings adopted (M6).
- [ACT-01](tasks/ACT-01.md) — [ACT-01] Implement account unlock action (M3).
- [ACT-02](tasks/ACT-02.md) — [ACT-02] Implement account password reset action (M4).
- [ACT-03](tasks/ACT-03.md) — [ACT-03] Implement account password change action (M4).
- [ACT-04](tasks/ACT-04.md) — [ACT-04] Implement object restore action (M7).
- [ACT-05](tasks/ACT-05.md) — [ACT-05] Implement directory server move action (M7).
- [ACT-06](tasks/ACT-06.md) — [ACT-06] Implement operation master role transfer action (M7).
- [ACT-07](tasks/ACT-07.md) — [ACT-07] Implement operation master role seize action (M7).
- [ACT-08](tasks/ACT-08.md) — [ACT-08] Implement domain functional level set action (M7).
- [ACT-09](tasks/ACT-09.md) — [ACT-09] Implement forest functional level set action (M7).
- [ACT-10](tasks/ACT-10.md) — [ACT-10] Implement optional feature enable action (M7).
- [ACT-11](tasks/ACT-11.md) — [ACT-11] Implement optional feature disable action (M7).
- [ACT-12](tasks/ACT-12.md) — [ACT-12] Implement object sync action (M7).
- [ACT-13](tasks/ACT-13.md) — [ACT-13] Implement service account migration start action (M7).
- [ACT-14](tasks/ACT-14.md) — [ACT-14] Implement service account migration complete action (M7).
- [ACT-15](tasks/ACT-15.md) — [ACT-15] Implement service account migration undo action (M7).
- [ACT-16](tasks/ACT-16.md) — [ACT-16] Implement service account migration reset action (M7).
- [CORE-14](tasks/CORE-14.md) — [CORE-14] Enable qualified cross-domain moves on the generic object lifecycle (M7).
- [REL-00](tasks/REL-00.md) — [REL-00] Verify the first qualified subset for an experimental provider release (M3).
- [REL-01](tasks/REL-01.md) — [REL-01] Audit all included capabilities and client/server qualification evidence (M8).
- [REL-02](tasks/REL-02.md) — [REL-02] Verify public examples, generated templates and user recovery guidance (M8).
- [REL-03](tasks/REL-03.md) — [REL-03] Publish reproducible signed provider release artifacts after qualification (M8).

## Execution conventions

Use namespace `activedirectory-provider:v1`; type/priority/area/component label dimensions and milestone mappings are specified in [metadata](metadata.json). Existing actual feature issues remain context unless explicitly a direct blocker. Native parents and blockers are required, not just body links. Readiness is computed from unresolved blockers; no due dates or synthetic blocked labels are assigned. Epic acceptance remains independent of child closure. Preserve human edits and reconcile both open and closed identities before writes.
