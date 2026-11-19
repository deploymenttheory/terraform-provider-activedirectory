# Organizational unit acceptance

The feature suite runs actual Terraform against the selected lab DC. It registers the OU resource,
single/plural data sources and list resource over production SDK clients. A separately authenticated
SDK observer checks fixture ownership, stable GUIDs and deletion. Neither client mocks nor the
PowerShell setup code participate in the provider's directory operations.

Use the authorization and protected authentication inputs in [the testing guide](testing.md).
Additionally, prepare an empty, unprotected OU with the SDK's marker-owned directory fixture:

| Input | Required value |
| --- | --- |
| `AD_ACCEPTANCE_PARENT_DN` | `OU=sdk-portable-<run-id>,<partition>` |
| `AD_ACCEPTANCE_PARENT_GUID` | Independently observed GUID of that OU |
| Parent description | `sdk-portable-parent:<run-id>` |

The SDK setup function is `New-PortableDirectoryFixture` from
[`scripts/lab/portable-directory-fixture.ps1`](https://github.com/deploymenttheory/go-sdk-activedirectory/blob/v0.2.0/scripts/lab/portable-directory-fixture.ps1).
It requires the selected lab's verified TLS fixture. Use a unique run identity for each invocation;
do not reuse an unresolved fixture after a failed or uncertain operation.

The feature-local [directory fixture](../../internal/services/resources/organizational_units/organizational_unit/tests/fixtures/directory.ps1)
provides `New-ProviderOuPermissionFixture` and `Remove-ProviderOuChildren`. Dot-source it on the owned
DC alongside the SDK TLS/directory fixture functions and the SDK's
[`kerberos-write-fixture.ps1`](https://github.com/deploymenttheory/go-sdk-activedirectory/blob/v0.2.0/scripts/lab/kerberos-write-fixture.ps1).
For the permission case, first use the SDK's `kerberos-account.ps1 -Action Prepare` to create an
ordinary marker-owned account. This creates no privileged membership and changes no domain policy.
Pass that account's run ID, GUID and the existing protected setup credential to
`New-ProviderOuPermissionFixture`, together with the verified parent GUID and client platform.
The function creates one owned OU and denies only description read/write to that account.
It verifies the entire ACL, preserving all existing ACEs and requiring canonical ordering.

Supply the returned fixture identity and ordinary account through process environment:

| Input | Value |
| --- | --- |
| `AD_ACCEPTANCE_HIDDEN_GUID` | Permission fixture GUID |
| `AD_ACCEPTANCE_HIDDEN_NAME` | Permission fixture name |
| `TF_VAR_reader_upn` | Ordinary fixture account UPN |
| `TF_VAR_reader_password` | Fixture account password from protected storage |

The Terraform password variable is sensitive and ephemeral. Never put credentials in command
arguments, configuration fixtures or evidence files. The main provider and independent observer
use the explicitly selected setup identity; the permission test switches to an isolated provider
alias with the ordinary account for denied-write and hidden-read checks.

Run one case per invocation, using the selected package or a checksum-pinned test binary with the
shared acceptance scripts:

```shell
go test ./internal/services/resources/organizational_units/organizational_unit \
  -run '^TestAccOrganizationalUnitLifecycle$' -count=1 -timeout=15m
```

| Test | Behavior checked |
| --- | --- |
| `TestAccOrganizationalUnitLifecycle` | CRUD, stable GUID through rename/move, no-change plans, GUID/identity import, generated configuration, both data sources, list query and explicit clear |
| `TestAccOrganizationalUnitProtectedDelete` | Protected delete fails with state retained; explicit unprotection permits deletion |
| `TestAccOrganizationalUnitNonemptyDelete` | Nonempty leaf delete fails; independently owned child must be retired explicitly |
| `TestAccOrganizationalUnitDeniedAndHidden` | Denied property update retains GUID/value; a hidden managed value is preserved while a fresh data lookup returns null; administrator confirms no property change |

On failure, let the test process retire before cleanup. `Remove-ProviderOuChildren` verifies the
entire bounded descendant inventory before performing leaf-only cleanup of matching GUIDs and
markers. It refuses unowned/moved objects. Then remove the empty outer OU, retire the ordinary
account with `Remove-KerberosDirectoryFixture`, and retire the TLS fixture with its SDK helper.
Verify absence independently and restore the original VM/client profile. Do not treat an exception
or successful delete submission as cleanup proof.

Record the provider revision, tool/binary hashes, Windows/Linux amd64 client, selected server
version, transport, each test result and cleanup receipt. Repeat applicable cases against Server
2016, 2019, 2022 and 2025. Keep bulk logs local; summarize the verified matrix in the implementation PR.
