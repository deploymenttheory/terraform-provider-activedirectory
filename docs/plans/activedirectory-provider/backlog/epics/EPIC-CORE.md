<!-- weave-backlog:activedirectory-provider:v1:EPIC-CORE -->
Backlog ID: `EPIC-CORE` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Core objects, accounts and individual relationships reconcile across the applicable matrix, including separately gated cross-domain moves.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [CORE-01](../tasks/CORE-01.md) — Implement organizational unit lifecycle and discovery; provider; M3.
- [ ] [CORE-02](../tasks/CORE-02.md) — Implement user lifecycle and discovery; provider; M3.
- [ ] [CORE-03](../tasks/CORE-03.md) — Implement group lifecycle and discovery; provider; M4.
- [ ] [CORE-04](../tasks/CORE-04.md) — Implement computer lifecycle and discovery; provider; M4.
- [ ] [CORE-05](../tasks/CORE-05.md) — Implement object lifecycle and discovery; provider; M4.
- [ ] [CORE-06](../tasks/CORE-06.md) — Implement service account lifecycle and discovery; provider; M4.
- [ ] [CORE-07](../tasks/CORE-07.md) — Implement account settings adopted; provider; M4.
- [ ] [CORE-08](../tasks/CORE-08.md) — Implement group member lifecycle and discovery; provider; M4.
- [ ] [CORE-09](../tasks/CORE-09.md) — Implement computer service account lifecycle and discovery; provider; M4.
- [ ] [CORE-10](../tasks/CORE-10.md) — Implement group members data; provider; M4.
- [ ] [CORE-11](../tasks/CORE-11.md) — Implement principal group membership data; provider; M4.
- [ ] [CORE-12](../tasks/CORE-12.md) — Implement account authorization groups data; provider; M4.
- [ ] [CORE-13](../tasks/CORE-13.md) — Implement account search data; provider; M4.
- [ ] [CORE-14](../tasks/CORE-14.md) — Enable qualified cross-domain moves on the generic object lifecycle; provider; M7.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M7; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P1 — required planned capability.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
