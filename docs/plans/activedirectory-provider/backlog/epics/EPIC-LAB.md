<!-- weave-backlog:activedirectory-provider:v1:EPIC-LAB -->
Backlog ID: `EPIC-LAB` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Authorized, isolated runners and required versioned fixtures support independently verified acceptance.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [LAB-01](../tasks/LAB-01.md) — Inventory versioned lab targets and acceptance fixture requirements; infrastructure; M1.
- [ ] [LAB-02](../tasks/LAB-02.md) — Provision safe Linux and Windows acceptance connectivity; infrastructure; M2.
- [ ] [LAB-03](../tasks/LAB-03.md) — Provision feature-specific advanced and destructive lab fixtures; infrastructure; M4.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M4; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P0 — foundation/integrity/release dependency.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
