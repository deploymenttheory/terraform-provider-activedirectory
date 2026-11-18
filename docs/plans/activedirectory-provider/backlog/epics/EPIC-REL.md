<!-- weave-backlog:activedirectory-provider:v1:EPIC-REL -->
Backlog ID: `EPIC-REL` · Target: `deploymenttheory/terraform-provider-activedirectory` · Product baseline: [INDEX](../INDEX.md)

## Outcome

Only reproducibly built, documented and evidenced provider capabilities reach release publication.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Use the reference provider structure, shared public SDK contracts, independent ownership and explicit capability gating. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [REL-00](../tasks/REL-00.md) — Verify the first qualified subset for an experimental provider release; verification; M3.
- [ ] [REL-01](../tasks/REL-01.md) — Audit all included capabilities and client/server qualification evidence; verification; M8.
- [ ] [REL-02](../tasks/REL-02.md) — Verify public examples, generated templates and user recovery guidance; documentation; M8.
- [ ] [REL-03](../tasks/REL-03.md) — Publish reproducible signed provider release artifacts after qualification; infrastructure; M8.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M8; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P0 — foundation/integrity/release dependency.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
