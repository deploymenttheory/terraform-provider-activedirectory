# Active Directory provider approval package

Status: implementation approach agreed on 2026-11-18; the concrete backlog below is prepared for approval, not published. No provider feature, SDK portability implementation, lab run or release is claimed by this documentation PR.

## Review order

1. [Capability matrix](capability-matrix.md): all 151 SDK operations, 145 included and six agreed exclusions.
2. [Terraform type catalog](terraform-types.md): 162 proposed types owned by 71 feature tickets.
3. [Architecture](architecture.md): the selected portable design and lifecycle/security contracts.
4. [Implementation sequence](implementation-plan.md): milestones, direct blockers and verification.
5. [Exact publication preview](backlog/preview.md): full bodies, repositories, metadata, native relationships and ready frontier.
6. [Research and provenance](research.md): what was inspected and what remains unqualified.

The preview contains 98 implementation tasks, 10 epics and two baseline indices across `deploymenttheory/terraform-provider-activedirectory` and `deploymenttheory/go-sdk-activedirectory`. No issues, labels, milestones or native relationships have been published.

## Product rules

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

These approved provider decisions explicitly supersede the SDK handoff's 2026-11-17 instruction to postpone provider work. They do not weaken the SDK v1 release gate or rewrite retained evidence.

## Approval and execution boundary

The user approved the implementation approach. Approving an unseen ticket breakdown is a separate matter: the installed [plan-to-github-issues skill](https://github.com/weaveplatform/ai-agent-skills/blob/7d354aedf8e36e9aab8a3e80d0581f9e4890420b/skills/plan-to-github-issues/SKILL.md) requires approval of the exact preview before metadata/issue writes. Review [preview.md](backlog/preview.md) and its content-hash manifest. Approval covers the listed creates and native relationships, with stable draft links resolved to published issue URLs.

After approval, re-read both repositories, reconcile identities, publish only missing approved objects and verify native parents/blockers. Then implement ready tickets using their delivery gates. Opening or merging this documentation PR alone does not mark feature implementation complete or authorize a release.

## Local verification

Run `python docs/plans/activedirectory-provider/backlog/validate.py` from the provider root. It checks inventory coverage, unique type names/IDs, complete bodies, local links, digests, metadata, milestones and the acyclic dependency graph. It does not contact the lab or publish GitHub work.
