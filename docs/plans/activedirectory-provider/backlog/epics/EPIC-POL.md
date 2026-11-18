<!-- weave-backlog:activedirectory-provider:v1:EPIC-POL -->
Backlog ID: `EPIC-POL` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Policy, descriptor, claims and resource-property management preserves security and ownership boundaries.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [POL-01](../tasks/POL-01.md) — Implement fine grained password policy lifecycle and discovery; provider; M5.
- [ ] [POL-02](../tasks/POL-02.md) — Implement fine grained password policy subject lifecycle and discovery; provider; M5.
- [ ] [POL-03](../tasks/POL-03.md) — Implement domain password policy adopted; provider; M5.
- [ ] [POL-04](../tasks/POL-04.md) — Implement user resultant password policy data; provider; M5.
- [ ] [POL-05](../tasks/POL-05.md) — Implement authentication policy lifecycle and discovery; provider; M5.
- [ ] [POL-06](../tasks/POL-06.md) — Implement authentication policy silo lifecycle and discovery; provider; M5.
- [ ] [POL-07](../tasks/POL-07.md) — Implement authentication policy silo access lifecycle and discovery; provider; M5.
- [ ] [POL-08](../tasks/POL-08.md) — Implement account authentication assignment adopted; provider; M5.
- [ ] [POL-09](../tasks/POL-09.md) — Implement central access rule lifecycle and discovery; provider; M5.
- [ ] [POL-10](../tasks/POL-10.md) — Implement central access policy lifecycle and discovery; provider; M5.
- [ ] [POL-11](../tasks/POL-11.md) — Implement central access policy member lifecycle and discovery; provider; M5.
- [ ] [POL-12](../tasks/POL-12.md) — Implement claim type lifecycle and discovery; provider; M5.
- [ ] [POL-13](../tasks/POL-13.md) — Implement claim transform policy lifecycle and discovery; provider; M5.
- [ ] [POL-14](../tasks/POL-14.md) — Implement claim transform link lifecycle and discovery; provider; M5.
- [ ] [POL-15](../tasks/POL-15.md) — Implement resource property lifecycle and discovery; provider; M5.
- [ ] [POL-16](../tasks/POL-16.md) — Implement resource property list lifecycle and discovery; provider; M5.
- [ ] [POL-17](../tasks/POL-17.md) — Implement resource property list member lifecycle and discovery; provider; M5.
- [ ] [POL-18](../tasks/POL-18.md) — Implement resource property value types data; provider; M5.
- [ ] [POL-19](../tasks/POL-19.md) — Implement authentication policy expression data; provider; M5.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M5; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P1 — required planned capability.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
