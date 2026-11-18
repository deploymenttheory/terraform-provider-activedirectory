# Implementation sequence and delivery gates

This is the approved implementation approach with its concrete ticket decomposition. Start provider development before full SDK v1 readiness, but release no feature before its SDK/provider evidence exists. The [publication preview](backlog/preview.md) must be approved before creating GitHub metadata or issues.

## Outcome milestones

| Milestone | Exit conditions |
| --- | --- |
| M1 — Provider foundation | Root Framework module, isolated registries/mocks, schema and docs tooling, mandatory CI and reproducible build commands exist. No live feature qualification is claimed. |
| M2 — Portable authenticated directory foundation | Portable LDAP, verified TLS, Kerberos CBT, SASL signing/sealing, outcome contracts, client integration and Linux/Windows lab connectivity pass their acceptance gates. |
| M3 — Four-surface reference slice | OU and user resource/data/list lifecycles and the unlock action have unit, docs/examples and applicable eight-cell live evidence; no-change and failure recovery are demonstrated. |
| M4 — Core directory management | Core objects, supported managed service accounts, account settings and individual relationships meet the same delivery bar. |
| M5 — Policies and access control | Password, authentication, central access, claims and resource property families pass descriptor/condition, privilege and relationship qualification. |
| M6 — Topology and observations | Topology, adopted settings, discovery and portable diagnostics cover the applicable client/server matrix without incomplete-read data loss. |
| M7 — Advanced directory actions | Explicitly acknowledged high-impact actions are qualified in isolated/restorable fixtures with outcome and recovery evidence. |
| M8 — Provider release qualification | All included surfaces, examples/templates, dependency and evidence audits, release artifacts and security checks pass; macOS live evidence remains explicitly unclaimed. |

M1 scaffold, CI, docs tooling and lab inventory can proceed independently. M2 establishes the shared portable/authenticated path before the M3 four-surface slice can qualify. Group related CRUD/data/list work into coherent feature tickets; actions and integrated verification have separate ownership. M4 core service-account work requires the advanced fixture task, so LAB-03 starts in M4 rather than being delayed until policy delivery. Later feature milestones are delivery gates, not artificial dependency edges. Cross-domain moves remain CORE-14/M7.

REL-00 provides the first experimental, qualified-subset release gate in M3. Later releases can add independently qualified surfaces with the same per-feature rules and an explicit release manifest. REL-01–03 are the final complete-programme audit/release gate, not a requirement to withhold every earlier release until all 145 included operations are delivered. Actual release publication remains separately authorized.

## Work order

Use direct blockers in [metadata.json](backlog/metadata.json), including cross-repository issue links. Supporting/transitive dependencies are retained in bodies and removed from native direct edges. Parents are hierarchy, not blockers. For the snapshot, ready tasks are:

- [FND-01](backlog/tasks/FND-01.md)
- [LAB-01](backlog/tasks/LAB-01.md)

Existing SDK #135/#140/#157 and #35 are explicit prerequisites where their contracts are genuinely required. Other SDK feature issues are contextual and do not impose unrelated variant closure. Do not make SDK #57/all-v1 qualification a provider-wide blocker. Fresh remote states determine readiness at publication and execution time.

## Verification matrix

| Dimension | Requirement |
| --- | --- |
| Live clients | Linux and Windows amd64 |
| Targets | Server 2016, 2019, 2022, 2025, per-feature applicability |
| macOS | Portable unit/build coverage; no live acceptance claim |
| arm64 | Reproducible builds and available native portable checks; live matrix remains explicitly unqualified |
| Authentication | Verified LDAPS/StartTLS simple and Kerberos with CBT; Kerberos LDAP signing and sealing |
| Object/relationship lifecycle | Create/adopt, Read, Update where supported, import, stable identity, repeated no-change plans, drift and Delete/relinquish |
| Failures | Denied/hidden/partial observations, normalization, concurrent changes, accepted change with failed readback, uncertain submission and independent reconciliation |
| List resources | Correct identity, paging/limits, cancellation/consumer stop, duplicate handling, generated config/import |
| Actions | Prerequisites, explicit target/risk acknowledgement, refused/accepted/uncertain effects and recovery |
| Security | Certificate/SPN/credential failures, CBT Always, signed/sealed traffic tamper/order failures and no downgrade |

Planned standard commands are `go test ./...`, `go vet ./...`, required read-only lint and `make generate-docs`. Foundation tickets supply `scripts/run-unit.ps1`, `scripts/run-acceptance.ps1` and documented Linux equivalents. Acceptance uses TF_ACC=1 and approved fixture/target configuration; no secret examples are committed. Every feature PR carries its unit/live evidence, public .tf files and .md.tmpl output; REL-01 audits rather than substitutes for those gates.

## Lab controls

Run lab PowerShell directly through approval. Never use RunAs, hidden elevation or alternative routes around prompts. Independent cleanup verification is required even when the operation itself passes. Use isolated/restorable fixtures for functional levels, optional-feature enablement, role seizure and migration. Read current lab state before assuming recorded profiles are active. Missing evidence blocks a claim.

## Execution completion

Each task needs a raised PR and passing required checks; applicable unit/lab evidence and every owned surface's examples/templates are part of the same ticket. A feature with unavailable acceptance cannot be called complete. Merges and release publication require separate authorization. Skills construct and publish work; they do not automatically implement/close epics.
