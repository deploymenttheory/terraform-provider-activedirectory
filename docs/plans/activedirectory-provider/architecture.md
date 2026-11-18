# Provider architecture and contracts

## Runtime and target

Terraform executes the provider on Linux, Windows or macOS; the target is AD DS on Windows Server. Live acceptance is Linux/Windows clients × Server 2016/2019/2022/2025 where applicable. macOS receives build/unit checks only. Use amd64 live qualification initially, with amd64/arm64 build and portable checks clearly distinguished. AD LDS and host-local/cloning operations are outside this programme.

The provider is a direct portable client. No gateway, WinRM/SSH runner or Windows helper is part of this architecture. Windows is not a prerequisite for an ordinary Linux Terraform run. SDK portability belongs in the SDK repository, with per-feature release blockers rather than a global wait for SDK v1.

## Module and feature layout

Use a root Go module and protocol V6 Plugin Framework provider. Reference pins are Go 1.26.0, Framework 1.19.0, plugin-go 0.31.0, plugin-testing 1.16.0, timeouts 0.7.0, validators 0.19.0 and plugin-docs 0.25.0; verify and lock the compatible set in FND-01. Terraform >=1.14 is the feature floor for list resources/actions.

```text
main.go
internal/provider/
  provider_factory.go
  resources.go / datasources.go / list_resources.go / actions.go
  core/                         # schema/configuration/registry
internal/client/                 # public SDK bundle and injectable interfaces
internal/mocks/                  # isolated registries/factories
internal/services/{resources,datasources,list-resources,actions}/<family>/<type>/
  resource.go | datasource.go | list_resource.go | action.go
  model.go
  crud.go | read.go | list.go | invoke.go
  construct.go / state.go
  validate.go / plan_modify.go / predicate.go  # as required
  provider_factories_test.go
  *_test.go / *_acceptance_test.go
  mocks/
  tests/responses/
  tests/terraform/{unit,acceptance}/
examples/{resources,data-sources,list-resources,actions}/activedirectory_<type>/
templates/{resources,data-sources,list-resources,actions}/<type>.md.tmpl
docs/{resources,data-sources,list-resources,actions}/
```

Schema lives with its surface declaration; Framework types and tfsdk-tagged models remain separate from request construction and state mapping. Keep feature-local tests/mocks and explicit production registrations. Each test factory registers only its required surfaces. Share configured clients and lifetime through all four Configure interfaces. Do not copy Graph version folders, OAuth/cloud clients, HTTP responders or OData filters into an LDAP provider.

## Portable SDK and security

The current SDK has internal session interfaces but public Dial selects Windows native code. Extend the SDK's public transport/configuration API; the provider never imports its internal packages. Keep portable LDAP, authenticated RPC, security descriptors and expression handling below the public SDK seam.

The blocking SDK dependency review starts with the SASL-capable jfjallid LDAP/Kerberos implementation, pins audited source revisions and validates protocol/crypto behavior before adoption. If it cannot pass the specified contract, the milestone remains blocked; do not silently replace signing/sealing with successful unprotected GSSAPI bind. Dependency licensing, maintenance, Go compatibility and vulnerability review are part of SDK-02.

Authentication is explicit: protected simple bind or Kerberos using password, keytab or supported file cache, explicit realm and service identity. Preserve an existing Windows integrated backend only when explicitly selected and supported. No implicit NTLM/simple/anonymous fallback is allowed. Observe actual negotiated protection.

Verified LDAPS and StartTLS require TLS >=1.2, trusted chain, hostname validation and explicit private CA support. TLS completes before credentials. Kerberos-over-TLS includes correct channel binding and mutual authentication; enforced CBT policy must be tested. Kerberos-over-LDAP requires the selected SASL integrity/confidentiality protection on subsequent messages, including ordering, framing and tamper verification. First usable release includes signing and sealing. A successful bind without required traffic protection is a failure.

Use established connection state and bounded operations. The current native SDK serializes a session; do not assume cancellation interrupts an in-flight synchronous native call. Retire compromised/cancelled sessions safely. Do not add blind mutation retries in provider middleware.

## Identity, lifecycle and state

Object resources use object GUID as canonical identity and import key. DN/name are mutable observations. A relationship resource uses the ordered GUID tuple plus list kind/direction where needed, with deterministic import serialization. List resource identity schemas match managed resource import/identity schemas exactly.

Resource models distinguish unknown configuration, unconfigured fields and observed values. Keep partial/unloaded/inaccessible attributes distinct from confirmed empty. On refresh, remove state only for confirmed absence of the selected identity; authorization, network and partial-result failures retain state and diagnostics.

After creation or other accepted changes, persist any known identity and accepted effects even when readback fails. Distinguish not attempted, rejected, accepted and uncertain mutations from subsequent readback. Read retries are bounded and feature-specific. A contradictory observation differs from one that cannot prove the result; neither permits blind write replay. No automatic rollback is assumed. Error messages redact credentials and sensitive raw attributes.

Same-domain rename/move are supported updates where exposed. Cross-domain generic-object moves are a later, separately qualified CORE-14 contract that can change identity/account observations; do not fabricate stable identity across it or silently enable it for all typed resources.

## Ownership and schemas

Typed resources own their explicitly managed attributes; generic objects must not co-manage those fields on the same object. Individual membership resources own exactly one edge and preserve unrelated members. Authoritative collection resources are deferred. Global conflict detection across separate Terraform states/workspaces is not promised; examples and diagnostics must explain conflicting ownership.

Domain, forest, account settings, domain password policy, authentication assignment and replication connection settings adopt existing objects. Create applies owned settings, import establishes identity and Delete relinquishes management without deleting/resetting underlying infrastructure or assignments. Explicit clearing is an Update operation, not an implicit relinquishment side effect.

A relationship Delete removes its owned link without deleting endpoints. High-impact changes are actions with explicit target and operation-specific acknowledgement; a transfer never silently becomes a seizure. Unsupported disable/downgrade operations fail before mutation.

Provider credentials and password values never enter state or logs. Resource password inputs use Framework write-only attributes and explicit change/version triggers; action secrets are handled only as invocation inputs. Sensitive marking alone is not an adequate state-protection mechanism.

## List resources, data sources and actions

List resources are import-oriented discovery for an implemented managed resource, not a synonym for every plural query. Honor paging, bounds, duplicate identities, cancellation and the consumer stop signal. Emit only usable identities and diagnose incomplete discovery. Use Framework identity support and test generated configuration/import. Prefer streaming where the SDK can supply pages; otherwise report bounded materialization honestly.

Data sources own no directory state. Scoped/diagnostic reads must state their selected endpoint, limits and observation semantics. Provider-local expression data does not launch a GUI. Actions validate configuration and prerequisites before side effects, expose useful progress without secrets, and preserve per-target/stage outcomes on partial failure.

## Documentation and delivery

Every resource, data source, list resource and action has feature-local unit fixtures, applicable live acceptance, public .tf examples, a .md.tmpl and generated docs. Resources include import examples; list resources include query/import discovery examples; actions include invocation examples and risk/recovery guidance. Follow the reference provider's naming and file separation, adapting semantics rather than copying Graph code.

No new ADR IDs are invented by this package. The agreed provider design is recorded here and in product rules R1–R12; existing SDK ADRs remain SDK contracts, not automatically provider decisions. Glossary terms are in the package glossary.
