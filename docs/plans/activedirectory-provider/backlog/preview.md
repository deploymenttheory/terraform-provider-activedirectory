# Exact backlog publication preview

Status: **prepared; awaiting approval of this exact set**. Destination repositories are `deploymenttheory/terraform-provider-activedirectory` and `deploymenttheory/go-sdk-activedirectory`; namespace `activedirectory-provider:v1`. No issue/metadata writes have occurred. The source approach is agreed; this full ticket/metadata/relationship set has not previously been shown for approval.

## Proposed changes

- Create two baseline indices, ten epics and 98 implementation tasks (110 total issues).
- Create only missing scoped labels and outcome milestones listed in [metadata.json](metadata.json), reusing genuine equivalents after fresh discovery. No due dates.
- Attach every task to its native epic and create 371 native direct-blocker edges, including existing SDK issue references. No issue closes, reopens, deletions or parent transfers are proposed.
- Reuse existing SDK feature tickets as supporting context; direct existing blockers are separately enumerated in metadata.
- Existing provider implementation issues: none observed. Open Dependabot PRs are unrelated. Reconcile all-state identities before publication; any overlap/conflict changes the preview rather than being silently overwritten.

## Requirement coverage and full bodies

Read [provider baseline](INDEX.md), [SDK baseline](SDK-INDEX.md), [all operation dispositions](../capability-matrix.md), [exact type catalog](../terraform-types.md), [architecture](../architecture.md), and [sequence](../implementation-plan.md). Every implementation ticket below has the final eight-section body and proposed PR title.

| Stable ID | Target | Gate | Full approved-draft body |
| --- | --- | --- | --- |
| INDEX | deploymenttheory/terraform-provider-activedirectory | M8 | [Active Directory provider: implementation backlog and product baseline](INDEX.md) |
| INDEX | deploymenttheory/go-sdk-activedirectory | M7 | [AD SDK provider portability: implementation backlog and product baseline](SDK-INDEX.md) |
| EPIC-FND | deploymenttheory/terraform-provider-activedirectory | M2 | [[Epic FND] Provider foundation](epics/EPIC-FND.md) |
| EPIC-SDK | deploymenttheory/go-sdk-activedirectory | M7 | [[Epic SDK] Portable SDK foundation and capabilities](epics/EPIC-SDK.md) |
| EPIC-LAB | deploymenttheory/terraform-provider-activedirectory | M4 | [[Epic LAB] Acceptance infrastructure](epics/EPIC-LAB.md) |
| EPIC-CORE | deploymenttheory/terraform-provider-activedirectory | M7 | [[Epic CORE] Core directory management](epics/EPIC-CORE.md) |
| EPIC-POL | deploymenttheory/terraform-provider-activedirectory | M5 | [[Epic POL] Policies and access control](epics/EPIC-POL.md) |
| EPIC-TOPO | deploymenttheory/terraform-provider-activedirectory | M6 | [[Epic TOPO] Replication topology](epics/EPIC-TOPO.md) |
| EPIC-READ | deploymenttheory/terraform-provider-activedirectory | M7 | [[Epic READ] Discovery and observations](epics/EPIC-READ.md) |
| EPIC-DIR | deploymenttheory/terraform-provider-activedirectory | M6 | [[Epic DIR] Adopted directory settings](epics/EPIC-DIR.md) |
| EPIC-ACT | deploymenttheory/terraform-provider-activedirectory | M7 | [[Epic ACT] Explicit directory actions](epics/EPIC-ACT.md) |
| EPIC-REL | deploymenttheory/terraform-provider-activedirectory | M8 | [[Epic REL] Integrated release qualification](epics/EPIC-REL.md) |
| SDK-01 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-01] Expose portable backend and authentication contracts without leaking internal packages](tasks/SDK-01.md) |
| SDK-02 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-02] Implement portable LDAP sessions, controls and binary-safe operations](tasks/SDK-02.md) |
| SDK-03 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-03] Implement verified LDAPS and StartTLS with measured channel protection](tasks/SDK-03.md) |
| SDK-04 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-04] Implement portable Kerberos authentication and TLS channel binding](tasks/SDK-04.md) |
| SDK-05 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-05] Implement and qualify Kerberos SASL signing and sealing](tasks/SDK-05.md) |
| SDK-06 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-06] Implement portable security descriptor parsing, editing and encoding](tasks/SDK-06.md) |
| SDK-07 | deploymenttheory/go-sdk-activedirectory | M6 | [[SDK-07] Implement portable authenticated replication diagnostics](tasks/SDK-07.md) |
| SDK-08 | deploymenttheory/go-sdk-activedirectory | M5 | [[SDK-08] Implement portable conditional expressions and descriptor conversion](tasks/SDK-08.md) |
| SDK-09 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-09] Preserve portable write outcomes, normalization and reconciliation contracts](tasks/SDK-09.md) |
| SDK-10 | deploymenttheory/go-sdk-activedirectory | M6 | [[SDK-10] Complete portable explicit-domain discovery and topology reads](tasks/SDK-10.md) |
| SDK-11 | deploymenttheory/go-sdk-activedirectory | M2 | [[SDK-11] Qualify the portable authenticated backend for provider use](tasks/SDK-11.md) |
| SDK-12 | deploymenttheory/go-sdk-activedirectory | M7 | [[SDK-12] Qualify portable cross-domain move and delegation behavior](tasks/SDK-12.md) |
| FND-01 | deploymenttheory/terraform-provider-activedirectory | M1 | [[FND-01] Create the root Terraform Plugin Framework provider module](tasks/FND-01.md) |
| FND-02 | deploymenttheory/terraform-provider-activedirectory | M1 | [[FND-02] Add explicit production registrations and isolated provider factories](tasks/FND-02.md) |
| FND-03 | deploymenttheory/terraform-provider-activedirectory | M2 | [[FND-03] Integrate portable SDK clients, credentials and session lifetime](tasks/FND-03.md) |
| FND-04 | deploymenttheory/terraform-provider-activedirectory | M2 | [[FND-04] Build SDK-boundary mocks and Terraform unit/acceptance harnesses](tasks/FND-04.md) |
| FND-05 | deploymenttheory/terraform-provider-activedirectory | M1 | [[FND-05] Add example and template-based documentation generation](tasks/FND-05.md) |
| FND-06 | deploymenttheory/terraform-provider-activedirectory | M1 | [[FND-06] Establish required CI and reproducible multi-platform builds](tasks/FND-06.md) |
| FND-07 | deploymenttheory/terraform-provider-activedirectory | M2 | [[FND-07] Implement stable identities, lifecycle diagnostics and bounded readback helpers](tasks/FND-07.md) |
| LAB-01 | deploymenttheory/terraform-provider-activedirectory | M1 | [[LAB-01] Inventory versioned lab targets and acceptance fixture requirements](tasks/LAB-01.md) |
| LAB-02 | deploymenttheory/terraform-provider-activedirectory | M2 | [[LAB-02] Provision safe Linux and Windows acceptance connectivity](tasks/LAB-02.md) |
| LAB-03 | deploymenttheory/terraform-provider-activedirectory | M4 | [[LAB-03] Provision feature-specific advanced and destructive lab fixtures](tasks/LAB-03.md) |
| CORE-01 | deploymenttheory/terraform-provider-activedirectory | M3 | [[CORE-01] Implement organizational unit lifecycle and discovery](tasks/CORE-01.md) |
| CORE-02 | deploymenttheory/terraform-provider-activedirectory | M3 | [[CORE-02] Implement user lifecycle and discovery](tasks/CORE-02.md) |
| CORE-03 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-03] Implement group lifecycle and discovery](tasks/CORE-03.md) |
| CORE-04 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-04] Implement computer lifecycle and discovery](tasks/CORE-04.md) |
| CORE-05 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-05] Implement object lifecycle and discovery](tasks/CORE-05.md) |
| CORE-06 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-06] Implement service account lifecycle and discovery](tasks/CORE-06.md) |
| CORE-07 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-07] Implement account settings adopted](tasks/CORE-07.md) |
| CORE-08 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-08] Implement group member lifecycle and discovery](tasks/CORE-08.md) |
| CORE-09 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-09] Implement computer service account lifecycle and discovery](tasks/CORE-09.md) |
| CORE-10 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-10] Implement group members data](tasks/CORE-10.md) |
| CORE-11 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-11] Implement principal group membership data](tasks/CORE-11.md) |
| CORE-12 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-12] Implement account authorization groups data](tasks/CORE-12.md) |
| CORE-13 | deploymenttheory/terraform-provider-activedirectory | M4 | [[CORE-13] Implement account search data](tasks/CORE-13.md) |
| POL-01 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-01] Implement fine grained password policy lifecycle and discovery](tasks/POL-01.md) |
| POL-02 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-02] Implement fine grained password policy subject lifecycle and discovery](tasks/POL-02.md) |
| POL-03 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-03] Implement domain password policy adopted](tasks/POL-03.md) |
| POL-04 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-04] Implement user resultant password policy data](tasks/POL-04.md) |
| POL-05 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-05] Implement authentication policy lifecycle and discovery](tasks/POL-05.md) |
| POL-06 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-06] Implement authentication policy silo lifecycle and discovery](tasks/POL-06.md) |
| POL-07 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-07] Implement authentication policy silo access lifecycle and discovery](tasks/POL-07.md) |
| POL-08 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-08] Implement account authentication assignment adopted](tasks/POL-08.md) |
| POL-09 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-09] Implement central access rule lifecycle and discovery](tasks/POL-09.md) |
| POL-10 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-10] Implement central access policy lifecycle and discovery](tasks/POL-10.md) |
| POL-11 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-11] Implement central access policy member lifecycle and discovery](tasks/POL-11.md) |
| POL-12 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-12] Implement claim type lifecycle and discovery](tasks/POL-12.md) |
| POL-13 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-13] Implement claim transform policy lifecycle and discovery](tasks/POL-13.md) |
| POL-14 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-14] Implement claim transform link lifecycle and discovery](tasks/POL-14.md) |
| POL-15 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-15] Implement resource property lifecycle and discovery](tasks/POL-15.md) |
| POL-16 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-16] Implement resource property list lifecycle and discovery](tasks/POL-16.md) |
| POL-17 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-17] Implement resource property list member lifecycle and discovery](tasks/POL-17.md) |
| POL-18 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-18] Implement resource property value types data](tasks/POL-18.md) |
| POL-19 | deploymenttheory/terraform-provider-activedirectory | M5 | [[POL-19] Implement authentication policy expression data](tasks/POL-19.md) |
| TOPO-01 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-01] Implement replication site lifecycle and discovery](tasks/TOPO-01.md) |
| TOPO-02 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-02] Implement replication site link lifecycle and discovery](tasks/TOPO-02.md) |
| TOPO-03 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-03] Implement replication site link bridge lifecycle and discovery](tasks/TOPO-03.md) |
| TOPO-04 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-04] Implement replication subnet lifecycle and discovery](tasks/TOPO-04.md) |
| TOPO-05 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-05] Implement replication connection settings adopted](tasks/TOPO-05.md) |
| TOPO-06 | deploymenttheory/terraform-provider-activedirectory | M6 | [[TOPO-06] Implement rodc password replication policy member lifecycle and discovery](tasks/TOPO-06.md) |
| READ-01 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-01] Implement domain data](tasks/READ-01.md) |
| READ-02 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-02] Implement forest data](tasks/READ-02.md) |
| READ-03 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-03] Implement domain controller data](tasks/READ-03.md) |
| READ-04 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-04] Implement trust data](tasks/READ-04.md) |
| READ-05 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-05] Implement root dse data](tasks/READ-05.md) |
| READ-06 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-06] Implement optional features data](tasks/READ-06.md) |
| READ-07 | deploymenttheory/terraform-provider-activedirectory | M7 | [[READ-07] Implement deleted objects data](tasks/READ-07.md) |
| READ-08 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-08] Implement rodc password replication policy data](tasks/READ-08.md) |
| READ-09 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-09] Implement account resultant password replication policy data](tasks/READ-09.md) |
| READ-10 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-10] Implement rodc password replication policy usage data](tasks/READ-10.md) |
| READ-11 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-11] Implement replication failures data](tasks/READ-11.md) |
| READ-12 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-12] Implement replication partner metadata data](tasks/READ-12.md) |
| READ-13 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-13] Implement replication queue data](tasks/READ-13.md) |
| READ-14 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-14] Implement replication attribute metadata data](tasks/READ-14.md) |
| READ-15 | deploymenttheory/terraform-provider-activedirectory | M6 | [[READ-15] Implement replication up to dateness vectors data](tasks/READ-15.md) |
| DIR-01 | deploymenttheory/terraform-provider-activedirectory | M6 | [[DIR-01] Implement domain settings adopted](tasks/DIR-01.md) |
| DIR-02 | deploymenttheory/terraform-provider-activedirectory | M6 | [[DIR-02] Implement forest settings adopted](tasks/DIR-02.md) |
| ACT-01 | deploymenttheory/terraform-provider-activedirectory | M3 | [[ACT-01] Implement account unlock action](tasks/ACT-01.md) |
| ACT-02 | deploymenttheory/terraform-provider-activedirectory | M4 | [[ACT-02] Implement account password reset action](tasks/ACT-02.md) |
| ACT-03 | deploymenttheory/terraform-provider-activedirectory | M4 | [[ACT-03] Implement account password change action](tasks/ACT-03.md) |
| ACT-04 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-04] Implement object restore action](tasks/ACT-04.md) |
| ACT-05 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-05] Implement directory server move action](tasks/ACT-05.md) |
| ACT-06 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-06] Implement operation master role transfer action](tasks/ACT-06.md) |
| ACT-07 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-07] Implement operation master role seize action](tasks/ACT-07.md) |
| ACT-08 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-08] Implement domain functional level set action](tasks/ACT-08.md) |
| ACT-09 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-09] Implement forest functional level set action](tasks/ACT-09.md) |
| ACT-10 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-10] Implement optional feature enable action](tasks/ACT-10.md) |
| ACT-11 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-11] Implement optional feature disable action](tasks/ACT-11.md) |
| ACT-12 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-12] Implement object sync action](tasks/ACT-12.md) |
| ACT-13 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-13] Implement service account migration start action](tasks/ACT-13.md) |
| ACT-14 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-14] Implement service account migration complete action](tasks/ACT-14.md) |
| ACT-15 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-15] Implement service account migration undo action](tasks/ACT-15.md) |
| ACT-16 | deploymenttheory/terraform-provider-activedirectory | M7 | [[ACT-16] Implement service account migration reset action](tasks/ACT-16.md) |
| CORE-14 | deploymenttheory/terraform-provider-activedirectory | M7 | [[CORE-14] Enable qualified cross-domain moves on the generic object lifecycle](tasks/CORE-14.md) |
| REL-00 | deploymenttheory/terraform-provider-activedirectory | M3 | [[REL-00] Verify the first qualified subset for an experimental provider release](tasks/REL-00.md) |
| REL-01 | deploymenttheory/terraform-provider-activedirectory | M8 | [[REL-01] Audit all included capabilities and client/server qualification evidence](tasks/REL-01.md) |
| REL-02 | deploymenttheory/terraform-provider-activedirectory | M8 | [[REL-02] Verify public examples, generated templates and user recovery guidance](tasks/REL-02.md) |
| REL-03 | deploymenttheory/terraform-provider-activedirectory | M8 | [[REL-03] Publish reproducible signed provider release artifacts after qualification](tasks/REL-03.md) |

## Labels, milestones, parents and direct blockers

All exact labels (name/color/description), milestone names/descriptions, per-issue assignments, parents and direct edges are in [metadata.json](metadata.json). Body digests lock each draft. [review-lock.json](review-lock.json) locks the complete reviewable source set. The validator checks coverage and graph consistency before approval/publication.

Ready-to-start frontier at this snapshot:

- `deploymenttheory/terraform-provider-activedirectory:FND-01`
- `deploymenttheory/terraform-provider-activedirectory:LAB-01`

There are no proposed updates, unchanged managed issues or obsolete managed tickets because this namespace has not been published. Existing external SDK issues are reused and never overwritten. Their fresh status/evidence is checked before publication; closure as not planned is not a satisfied contract.

## Publication substitutions and verification

Resolve links to the local baseline/epic/task bodies into the corresponding repository issue URLs after their stable-ID mappings exist. Resolve shared plan/source document links to the approved provider Git commit and retain anchors. These substitutions do not change scope or acceptance. Keep explicit cross-repository targets and exact identity markers.

Publish baseline/epics first, then tasks in blocker order, using exact body files. Native parent/blocker requests require database IDs, not issue numbers. Re-read bodies/labels/milestones/parents/blockers and compare with the approved set. Preserve a source digest, issue-ID map and append-only recovery journal; after a timeout, discover remotely before retrying. Do not label publication complete if native relationships are unsupported or unverified.

The installed skill requires approval of this concrete preview before any issue, label, milestone or relationship write. If approved source files or metadata change materially, regenerate the preview and obtain approval of the changed set. No publishing script or automatic GitHub write is invoked by this documentation package.
