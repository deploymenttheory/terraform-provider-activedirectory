<!-- weave-backlog:activedirectory-provider:v1:FND-01 -->
Backlog ID: `FND-01` · Target: `deploymenttheory/terraform-provider-activedirectory` · Epic: [EPIC-FND](../epics/EPIC-FND.md) · Product baseline: [INDEX](../INDEX.md)

## Outcome

Create the root Terraform Plugin Framework provider module. The current provider is a template, and SDK support/qualification must be established rather than inferred from a planned capability.

## Implementation

Replace template/workload guidance with a root Go module, provider entry point and version plumbing. Start from the verified Microsoft 365 reference pins (Go 1.26, Framework 1.19, plugin-go 0.31, plugin-testing 1.16, timeouts 0.7, validators 0.19, plugin-docs 0.25); verify compatibility and pin explicitly. Require Terraform 1.14 or newer for the four-surface provider. Use deploymenttheory/activedirectory and activedirectory_ names. SDKv2 provider implementation is excluded.

Scope owner: provider. Adjacent protocol, fixture and provider work remains owned by the linked predecessor tasks. All paths introduced here are planned paths, not existing implementation claims.

## Acceptance criteria

- [ ] Empty provider validates and serves protocol V6 schemas on supported build targets.
- [ ] Root module CI paths and README replace template/segraef/workload assumptions.
- [ ] Raised implementation PR; passing required CI; meaningful unit checks and applicable lab evidence (a purely offline deliverable must record why live tests are not applicable).


## Verification

Use the specific checks above plus repository validation appropriate to this deliverable. Foundation tickets establish go test ./..., go vet ./..., required lint, make generate-docs and scripts/run-acceptance.ps1 (with documented Linux equivalent). Demonstrate the owned behavior rather than add shallow tests mirroring file contents. Infrastructure must record verified setup and applicable live smoke/security evidence; purely offline documentation/scaffold work records an explicit live-test not-applicable rationale.

## Dependencies

Direct blockers:

None; ready to start after publication approval.

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

Milestone: M1 — Provider foundation.
Priority: P0 — foundation/integrity/release blocker.
Component / scope owner: provider; destination `deploymenttheory/terraform-provider-activedirectory`.
Source rationale: [implementation sequence](../../implementation-plan.md) and [capability mapping](../../capability-matrix.md), source revisions in [research](../../research.md).
Suggested PR title: `feat(fnd): create the root Terraform Plugin Framework provider module`.

## Architecture decision alignment

Follow the agreed [provider architecture](../../architecture.md), especially portable execution, public SDK boundaries, individual ownership and preserved outcomes. [Glossary](../../glossary.md) distinguishes execution runtime from directory target and observation from qualification. No provider ADR number is claimed; existing SDK ADRs remain applicable to SDK-owned contracts. See the linked baseline for the agreed decision summary.
