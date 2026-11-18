<!-- weave-backlog:activedirectory-provider:v1:EPIC-ACT -->
Backlog ID: `EPIC-ACT` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Imperative and high-impact workflows preserve intent, risk acknowledgement, and recovery observations.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [ACT-01](../tasks/ACT-01.md) — Implement account unlock action; provider; M3.
- [ ] [ACT-02](../tasks/ACT-02.md) — Implement account password reset action; provider; M4.
- [ ] [ACT-03](../tasks/ACT-03.md) — Implement account password change action; provider; M4.
- [ ] [ACT-04](../tasks/ACT-04.md) — Implement object restore action; provider; M7.
- [ ] [ACT-05](../tasks/ACT-05.md) — Implement directory server move action; provider; M7.
- [ ] [ACT-06](../tasks/ACT-06.md) — Implement operation master role transfer action; provider; M7.
- [ ] [ACT-07](../tasks/ACT-07.md) — Implement operation master role seize action; provider; M7.
- [ ] [ACT-08](../tasks/ACT-08.md) — Implement domain functional level set action; provider; M7.
- [ ] [ACT-09](../tasks/ACT-09.md) — Implement forest functional level set action; provider; M7.
- [ ] [ACT-10](../tasks/ACT-10.md) — Implement optional feature enable action; provider; M7.
- [ ] [ACT-11](../tasks/ACT-11.md) — Implement optional feature disable action; provider; M7.
- [ ] [ACT-12](../tasks/ACT-12.md) — Implement object sync action; provider; M7.
- [ ] [ACT-13](../tasks/ACT-13.md) — Implement service account migration start action; provider; M7.
- [ ] [ACT-14](../tasks/ACT-14.md) — Implement service account migration complete action; provider; M7.
- [ ] [ACT-15](../tasks/ACT-15.md) — Implement service account migration undo action; provider; M7.
- [ ] [ACT-16](../tasks/ACT-16.md) — Implement service account migration reset action; provider; M7.

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
