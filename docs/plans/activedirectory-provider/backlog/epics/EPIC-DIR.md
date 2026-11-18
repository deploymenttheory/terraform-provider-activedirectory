<!-- weave-backlog:activedirectory-provider:v1:EPIC-DIR -->
Backlog ID: `EPIC-DIR` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Domain and forest settings reconcile without lifecycle claims over infrastructure provisioning.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [DIR-01](../tasks/DIR-01.md) — Implement domain settings adopted; provider; M6.
- [ ] [DIR-02](../tasks/DIR-02.md) — Implement forest settings adopted; provider; M6.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M6; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P1 — required planned capability.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
