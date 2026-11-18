<!-- weave-backlog:activedirectory-provider:v1:ACT-12 -->
Backlog ID: `ACT-12` · Target: `deploymenttheory/terraform-provider-activedirectory` · Epic: [EPIC-ACT](../epics/EPIC-ACT.md) · Product baseline: [INDEX](../INDEX.md)

## Outcome

Implement object sync action. The current provider is a template, and SDK support/qualification must be established rather than inferred from a planned capability.

## Implementation

Implement the following public types: action activedirectory_object_sync. Synchronize one object between explicitly selected domain controllers using the SDK LDAP operation; preserve target scope and failure details. Keep plan-to-SDK construction and response-to-state mapping separate, with Framework models and the gold-standard feature-local layout. Provide explicit production registration, isolated factories, mocks, unit/acceptance .tf fixtures, public examples, .md.tmpl files and generated docs for every listed type.

Scope owner: provider. Adjacent protocol, fixture and provider work remains owned by the linked predecessor tasks. All paths introduced here are planned paths, not existing implementation claims.

## Acceptance criteria

- [ ] Unit tests cover validation, capability refusal, cancellation and accepted/rejected/uncertain outcomes.
- [ ] Applicable lab success/failure/recovery cases include explicit target selection, redaction, independent readback and fixture cleanup.
- [ ] Raised implementation PR; passing required CI; meaningful unit checks and applicable lab evidence (missing lab fixtures block feature completion).

- [ ] Every owned Terraform surface has public examples, feature-local fixtures, its .md.tmpl and regenerated docs.

## Verification

Planned paths: `internal/services/<surface>/replication/object_sync/` with `tests/terraform/unit/`, `tests/terraform/acceptance/` and isolated provider factories. Run `go test ./internal/services/...`, `go vet ./...`, required lint and `make generate-docs`; FND-04 supplies the selected-feature acceptance runner. Execute applicable feature tests with TF_ACC=1 on Linux and Windows against Server 2016/2019/2022/2025 after approved lab prechecks. Do not mark unsupported on a skipped/missing fixture. Retain client/server build, auth/protection mode, source/binary revision, fixture identity, observed effects and independently verified cleanup. Test macOS through portable unit/build checks only. These commands/paths become executable through foundation tickets; no current provider test result is claimed.

## Dependencies

Direct blockers:

- [FND-07](FND-07.md) — required predecessor: Implement stable identities, lifecycle diagnostics and bounded readback helpers.
- [FND-05](FND-05.md) — required predecessor: Add example and template-based documentation generation.
- [FND-06](FND-06.md) — required predecessor: Establish required CI and reproducible multi-platform builds.
- [SDK-11](SDK-11.md) — required predecessor: Qualify the portable authenticated backend for provider use.
- [LAB-03](LAB-03.md) — required predecessor: Provision feature-specific advanced and destructive lab fixtures.

Supporting contracts and existing context:


- [SDK #37](https://github.com/deploymenttheory/go-sdk-activedirectory/issues/37) — existing feature/qualification context, not a requirement to close every unrelated SDK variant before this provider ticket.

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

Milestone: M7 — Advanced directory actions.
Priority: P1 — required planned capability.
Component / scope owner: provider; destination `deploymenttheory/terraform-provider-activedirectory`.
Source rationale: [implementation sequence](../../implementation-plan.md) and [capability mapping](../../capability-matrix.md), source revisions in [research](../../research.md).
Suggested PR title: `feat(act): implement object sync action`.

## Architecture decision alignment

Follow the agreed [provider architecture](../../architecture.md), especially portable execution, public SDK boundaries, individual ownership and preserved outcomes. [Glossary](../../glossary.md) distinguishes execution runtime from directory target and observation from qualification. No provider ADR number is claimed; existing SDK ADRs remain applicable to SDK-owned contracts. See the linked baseline for the agreed decision summary.
