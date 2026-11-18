# Research and provenance

Inspected on 2026-11-18. These are source facts, not passing provider tests.

| Source | Observed revision |
| --- | --- |
| Provider base | `ac71bba26964c7a9b13e8b41ed35843ce07a9802` |
| AD SDK working branch | `90ec7951314b41d49dd6bc78ddb3196afbdcb76d` |
| Microsoft 365 reference checkout | `a4f90832f7c620c4b36ad4fdc1be3a12f90b8117` |
| Weaveplatform skill | `7d354aedf8e36e9aab8a3e80d0581f9e4890420b` |
| SDK inventory SHA-256 | `197b54025bc60006db6daec4c3852bb9e5c3c9cd8b533c126175cbbf26909455` |

## Existing repositories

The target is template-only: no Go module or implementation, root CI expects go.mod, workload/README suggests a different layout, release configuration uses terraform-module, and template documentation still contains segraef references. FND-01/FND-06 own replacement; this planning PR does not pretend to be an implemented provider.

All-state provider issue inspection found no actual implementation issues and no milestones; open Dependabot pull requests are not tickets. Existing labels are stock labels, dependencies and github_actions. All-state SDK title inspection found no transport/Kerberos/SASL/Linux portability ticket to adopt. Existing feature and qualification issues are retained as context, with selected genuine blockers explicitly represented in metadata. Re-read all-state remote identities and SDK statuses immediately before publication.

## Gold-standard source

The local reference is the nested checkout `terraform-provider-microsoft365/terraform-provider-microsoft365`, not its worktree parent. Verified examples are [group lifecycle](https://github.com/deploymenttheory/terraform-provider-microsoft365/tree/a4f90832f7c620c4b36ad4fdc1be3a12f90b8117/internal/services/resources/groups/graph_beta/group), [isolated core registry](https://github.com/deploymenttheory/terraform-provider-microsoft365/blob/a4f90832f7c620c4b36ad4fdc1be3a12f90b8117/internal/provider/core/registry.go), and [list-resource example](https://github.com/deploymenttheory/terraform-provider-microsoft365/tree/a4f90832f7c620c4b36ad4fdc1be3a12f90b8117/internal/services/list-resources/users/graph_beta/user). Observed pins are listed in architecture.md. Reuse factories, layout, construction/state separation, predicates, unit/acceptance fixtures and templates; do not copy Graph authentication or HTTP behavior.

## SDK boundaries and gaps

The [SDK readiness gate](https://github.com/deploymenttheory/go-sdk-activedirectory/blob/90ec7951314b41d49dd6bc78ddb3196afbdcb76d/docs/sdk-readiness.md) is unmet. The retained baseline reports 178/1,070 behavioral scenarios, 0/253 variants and 0/151 fully validated operations. Those aggregate counts do not qualify the required version matrix. The agreed new direction is per-feature provider development; SDK release claims remain governed by SDK evidence.

Public Dial selects native Windows behavior; non-Windows builds return unsupported for live native operations. Internal session interfaces are a useful seam, but the provider cannot import them. Native dependencies include WinLDAP/SSPI, security descriptor conversion, conditional ACE compilation and replication diagnostics. Sync-ADObject and FSMO operations in this SDK use LDAP; do not incorrectly label them inherently RPC.

Replication connection has read/search/update but no typed Create/Delete. Attribute metadata/up-to-dateness implementation is tracked by [SDK #35](https://github.com/deploymenttheory/go-sdk-activedirectory/issues/35), not assumed present. Ordinary account lifecycle accepts sMSA/gMSA and rejects dMSA creation; migration actions use separately provisioned fixtures. Use [#135](https://github.com/deploymenttheory/go-sdk-activedirectory/issues/135), [#140](https://github.com/deploymenttheory/go-sdk-activedirectory/issues/140), [#157](https://github.com/deploymenttheory/go-sdk-activedirectory/issues/157) for existing capability/outcome/normalization contracts rather than duplicate their issues.

## Security evidence

Upstream [go-ldap GSSAPI bind](https://github.com/go-ldap/ldap/blob/master/v3/bind.go) does not implement subsequent SASL security layers; its [adapter](https://github.com/go-ldap/ldap/blob/v3.4.15/v3/gssapi/client.go) cannot be assumed channel-binding capable. The [jfjallid bind implementation](https://github.com/jfjallid/ldap/blob/master/v3/bind.go) and [adapter](https://github.com/jfjallid/ldap/blob/master/v3/gssapi/client.go) expose SASL protection and channel-binding integration and are concrete audit candidates, not qualified dependencies. Resolve and pin immutable versions during SDK-02.

Microsoft distinguishes [LDAP signing](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-signing) from [channel binding](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-channel-binding). The owner explicitly requires SASL signing/sealing as well as TLS modes; a TLS-only milestone cannot substitute for the first usable release gate.

Framework [list resources](https://developer.hashicorp.com/terraform/plugin/framework/list-resources), [actions](https://developer.hashicorp.com/terraform/plugin/framework/actions) and [write-only attributes](https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-arguments) determine surface support and password/state handling. Verify pinned-tool documentation in foundation implementation rather than assuming latest dependencies are compatible.

## Lab facts and limits

Documented lab root is H:/ad-sdk-lab with Server 2016/2019/2022/2025 profile directories. Server 2025 base topology documents ad.sdk.test, two writable DNS/GC DCs and a member on an isolated Hyper-V switch (10.77.23.10/.11/.20), no host connection or gateway, and 2016 base functional levels. State-file access was denied during planning; current guest runtime state is not verified. No lab command was executed for this plan.

Linux connectivity, RODC/trust/child-domain, KDS/gMSA/dMSA and destructive fixture availability require LAB-01–03 work. Follow the newer canonical [coverage handoff](https://github.com/deploymenttheory/go-sdk-activedirectory/blob/90ec7951314b41d49dd6bc78ddb3196afbdcb76d/docs/coverage-handoff.md): raise direct PowerShell commands through approval; no RunAs/elevation or prompt bypass. Retained reference runs are not provider acceptance.

## Skill installation

Installed user-level Weaveplatform marketplace/plugin in both clients using the repository's documented commands:

```text
codex plugin marketplace add weaveplatform/ai-agent-skills
codex plugin add weaveplatform-skills@weaveplatform
claude plugin marketplace add weaveplatform/ai-agent-skills
claude plugin install weaveplatform-skills@weaveplatform --scope user
```

Read the installed plan-to-github-issues SKILL.md and all four references to construct this package. Plugin availability may require a new client turn/session. The skill is not a publishing approval: its exact-preview gate still applies. The design-system skill is installed as part of the plugin but is not applied to this provider.
