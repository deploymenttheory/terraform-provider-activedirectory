<!-- weave-backlog:activedirectory-provider:v1:EPIC-SDK -->
Backlog ID: `EPIC-SDK` · Target: `deploymenttheory/go-sdk-activedirectory` · Product baseline: [INDEX](../SDK-INDEX.md)

## Outcome

Portable authenticated directory protocols and feature-specific representations preserve SDK observations and mutation outcomes.

## Agreed behavior

Follow [architecture](../../architecture.md) and product rules R1–R12. Protocol/security behavior belongs in the SDK and requires portable qualification; no provider internal-package imports or Windows helper. Each child owns its tests/examples/templates and applicable evidence rather than deferring that work to release.

## Implementation tickets

- [ ] [SDK-01](../tasks/SDK-01.md) — Expose portable backend and authentication contracts without leaking internal packages; sdk; M2.
- [ ] [SDK-02](../tasks/SDK-02.md) — Implement portable LDAP sessions, controls and binary-safe operations; sdk; M2.
- [ ] [SDK-03](../tasks/SDK-03.md) — Implement verified LDAPS and StartTLS with measured channel protection; sdk; M2.
- [ ] [SDK-04](../tasks/SDK-04.md) — Implement portable Kerberos authentication and TLS channel binding; sdk; M2.
- [ ] [SDK-05](../tasks/SDK-05.md) — Implement and qualify Kerberos SASL signing and sealing; sdk; M2.
- [ ] [SDK-06](../tasks/SDK-06.md) — Implement portable security descriptor parsing, editing and encoding; sdk; M2.
- [ ] [SDK-07](../tasks/SDK-07.md) — Implement portable authenticated replication diagnostics; sdk; M6.
- [ ] [SDK-08](../tasks/SDK-08.md) — Implement portable conditional expressions and descriptor conversion; sdk; M5.
- [ ] [SDK-09](../tasks/SDK-09.md) — Preserve portable write outcomes, normalization and reconciliation contracts; sdk; M2.
- [ ] [SDK-10](../tasks/SDK-10.md) — Complete portable explicit-domain discovery and topology reads; sdk; M6.
- [ ] [SDK-11](../tasks/SDK-11.md) — Qualify the portable authenticated backend for provider use; verification; M2.
- [ ] [SDK-12](../tasks/SDK-12.md) — Qualify portable cross-domain move and delegation behavior; sdk; M7.

## Epic acceptance

- [ ] Demonstrate integrated behavior across children, including shared configuration, error boundaries and qualified prerequisites.
- [ ] Audit per-child raised PR, required CI and applicable evidence; child closure alone is insufficient.
- [ ] Verify no included behavior contradicts ownership, protection, excluded-host-operation or runtime claims.

## Planning

Milestone: M7; acceptance completes when the integrated area meets its final delivery gate. Children may span earlier gates. No due date is assigned.
Priority: P0 — foundation/integrity/release dependency.
Source rationale: [implementation plan](../../implementation-plan.md), [capability matrix](../../capability-matrix.md).

## Architecture decision alignment

The agreed provider design is [architecture.md](../../architecture.md), not an invented ADR number. Preserve existing SDK qualification and observation contracts; per-feature development does not claim full SDK v1 readiness.
