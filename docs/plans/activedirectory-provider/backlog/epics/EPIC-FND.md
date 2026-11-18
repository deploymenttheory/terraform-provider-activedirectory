<!-- weave-backlog:activedirectory-provider:v1:EPIC-FND -->
Backlog ID: `EPIC-FND` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

A configured, isolated, testable Framework provider supports all four surface types and shared lifecycle contracts.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [FND-01](../tasks/FND-01.md) — Create the root Terraform Plugin Framework provider module; provider; M1.
- [ ] [FND-02](../tasks/FND-02.md) — Add explicit production registrations and isolated provider factories; provider; M1.
- [ ] [FND-03](../tasks/FND-03.md) — Integrate portable SDK clients, credentials and session lifetime; provider; M2.
- [ ] [FND-04](../tasks/FND-04.md) — Build SDK-boundary mocks and Terraform unit/acceptance harnesses; verification; M2.
- [ ] [FND-05](../tasks/FND-05.md) — Add example and template-based documentation generation; documentation; M1.
- [ ] [FND-06](../tasks/FND-06.md) — Establish required CI and reproducible multi-platform builds; infrastructure; M1.
- [ ] [FND-07](../tasks/FND-07.md) — Implement stable identities, lifecycle diagnostics and bounded readback helpers; provider; M2.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M2; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P0 — foundation/integrity/release dependency.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
