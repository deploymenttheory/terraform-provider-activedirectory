<!-- weave-backlog:activedirectory-provider:v1:EPIC-READ -->
Backlog ID: `EPIC-READ` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Scoped reads and diagnostics preserve completeness and do not invent managed ownership.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [READ-01](../tasks/READ-01.md) — Implement domain data; provider; M6.
- [ ] [READ-02](../tasks/READ-02.md) — Implement forest data; provider; M6.
- [ ] [READ-03](../tasks/READ-03.md) — Implement domain controller data; provider; M6.
- [ ] [READ-04](../tasks/READ-04.md) — Implement trust data; provider; M6.
- [ ] [READ-05](../tasks/READ-05.md) — Implement root dse data; provider; M6.
- [ ] [READ-06](../tasks/READ-06.md) — Implement optional features data; provider; M6.
- [ ] [READ-07](../tasks/READ-07.md) — Implement deleted objects data; provider; M7.
- [ ] [READ-08](../tasks/READ-08.md) — Implement rodc password replication policy data; provider; M6.
- [ ] [READ-09](../tasks/READ-09.md) — Implement account resultant password replication policy data; provider; M6.
- [ ] [READ-10](../tasks/READ-10.md) — Implement rodc password replication policy usage data; provider; M6.
- [ ] [READ-11](../tasks/READ-11.md) — Implement replication failures data; provider; M6.
- [ ] [READ-12](../tasks/READ-12.md) — Implement replication partner metadata data; provider; M6.
- [ ] [READ-13](../tasks/READ-13.md) — Implement replication queue data; provider; M6.
- [ ] [READ-14](../tasks/READ-14.md) — Implement replication attribute metadata data; provider; M6.
- [ ] [READ-15](../tasks/READ-15.md) — Implement replication up to dateness vectors data; provider; M6.

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
