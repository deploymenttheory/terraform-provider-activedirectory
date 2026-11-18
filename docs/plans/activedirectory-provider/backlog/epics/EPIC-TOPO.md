<!-- weave-backlog:activedirectory-provider:v1:EPIC-TOPO -->
Backlog ID: `EPIC-TOPO` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Topology management, adopted connections and RODC relationships preserve referenced objects and partial outcomes.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [TOPO-01](../tasks/TOPO-01.md) — Implement replication site lifecycle and discovery; provider; M6.
- [ ] [TOPO-02](../tasks/TOPO-02.md) — Implement replication site link lifecycle and discovery; provider; M6.
- [ ] [TOPO-03](../tasks/TOPO-03.md) — Implement replication site link bridge lifecycle and discovery; provider; M6.
- [ ] [TOPO-04](../tasks/TOPO-04.md) — Implement replication subnet lifecycle and discovery; provider; M6.
- [ ] [TOPO-05](../tasks/TOPO-05.md) — Implement replication connection settings adopted; provider; M6.
- [ ] [TOPO-06](../tasks/TOPO-06.md) — Implement rodc password replication policy member lifecycle and discovery; provider; M6.

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
